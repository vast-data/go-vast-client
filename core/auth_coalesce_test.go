package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// tokenGate holds token requests inside the test server handler until the test
// releases them. This makes the "leader is still authorizing" window explicit
// instead of relying on a sleep inside the handler.
type tokenGate struct {
	reached chan struct{}
	release chan struct{}
	once    sync.Once
}

func newTokenGate() *tokenGate {
	return &tokenGate{
		reached: make(chan struct{}),
		release: make(chan struct{}),
	}
}

// enter is called from the handler. It reports that a request arrived and then
// blocks until the test calls releaseAll.
func (g *tokenGate) enter() {
	g.once.Do(func() { close(g.reached) })
	<-g.release
}

// waitReached blocks until the first token request reaches the handler, which
// means the leader goroutine is inside its HTTP call with authorizing == true.
func (g *tokenGate) waitReached() {
	<-g.reached
}

func (g *tokenGate) releaseAll() {
	close(g.release)
}

// followerParkTime gives freshly started follower goroutines time to block on
// authCond. Followers perform no I/O before parking, so this is generous.
const followerParkTime = 200 * time.Millisecond

func newCoalesceTestAuth(host string, port uint64) *JWTAuthenticator {
	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)
	return auth
}

func assertTokenAndGeneration(t *testing.T, auth *JWTAuthenticator, wantToken string, wantGeneration uint64) {
	t.Helper()

	auth.mu.RLock()
	defer auth.mu.RUnlock()

	if auth.Token.Access != wantToken {
		t.Errorf("access token = %q, want %q", auth.Token.Access, wantToken)
	}
	if auth.generation != wantGeneration {
		t.Errorf("generation = %d, want %d", auth.generation, wantGeneration)
	}
}

func writeTokenResponse(w http.ResponseWriter, access, refresh string) {
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"access":  access,
		"refresh": refresh,
	})
}

// TestNoRedundantAcquireWhileLeaderInFlight verifies that when many goroutines
// need a first token at once, only the leader performs the HTTP call and every
// follower reuses the token it obtained.
func TestNoRedundantAcquireWhileLeaderInFlight(t *testing.T) {
	var acquireCalls int32
	gate := newTokenGate()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token/" {
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		n := atomic.AddInt32(&acquireCalls, 1)
		gate.enter()
		writeTokenResponse(w, "access-"+strconv.Itoa(int(n)), "refresh-"+strconv.Itoa(int(n)))
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := newCoalesceTestAuth(host, port)

	const followers = 20
	results := make(chan error, followers+1)

	go func() { results <- auth.authorize() }()
	gate.waitReached() // Leader is inside its HTTP call

	for i := 0; i < followers; i++ {
		go func() { results <- auth.authorize() }()
	}
	time.Sleep(followerParkTime)
	gate.releaseAll()

	for i := 0; i < followers+1; i++ {
		if err := <-results; err != nil {
			t.Errorf("authorize failed: %v", err)
		}
	}

	if calls := atomic.LoadInt32(&acquireCalls); calls != 1 {
		t.Errorf("token acquisitions = %d, want 1 (redundant token requests!)", calls)
	}
	assertTokenAndGeneration(t, auth, "access-1", 1)
}

// TestNoRedundantRefreshWhileLeaderInFlight verifies the same coalescing for an
// already initialized authenticator, which is the case when a batch of requests
// fails with 401/403 and each handler asks for a refresh.
func TestNoRedundantRefreshWhileLeaderInFlight(t *testing.T) {
	var refreshCalls, acquireCalls int32
	gate := newTokenGate()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/token/refresh/":
			n := atomic.AddInt32(&refreshCalls, 1)
			gate.enter()
			writeTokenResponse(w, "refreshed-"+strconv.Itoa(int(n)), "refresh-"+strconv.Itoa(int(n)))
		case "/api/token/":
			atomic.AddInt32(&acquireCalls, 1)
			writeTokenResponse(w, "acquired", "refresh-acquired")
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := newCoalesceTestAuth(host, port)
	auth.Token = &jwtToken{Access: "stale-token", Refresh: "stale-refresh"}
	auth.setInitialized(true)

	const followers = 20
	results := make(chan error, followers+1)

	go func() { results <- auth.authorize() }()
	gate.waitReached()

	for i := 0; i < followers; i++ {
		go func() { results <- auth.authorize() }()
	}
	time.Sleep(followerParkTime)
	gate.releaseAll()

	for i := 0; i < followers+1; i++ {
		if err := <-results; err != nil {
			t.Errorf("authorize failed: %v", err)
		}
	}

	if calls := atomic.LoadInt32(&refreshCalls); calls != 1 {
		t.Errorf("token refreshes = %d, want 1 (redundant token refreshes!)", calls)
	}
	if calls := atomic.LoadInt32(&acquireCalls); calls != 0 {
		t.Errorf("token acquisitions = %d, want 0 (refresh token was still valid)", calls)
	}
	assertTokenAndGeneration(t, auth, "refreshed-1", 1)
}

// TestWaitersRetryWhenLeaderAuthorizeFails verifies that coalescing is keyed on
// a successful authorization: when the leader fails, exactly one waiter retries
// and the rest reuse its token. A failing leader must not silently satisfy the
// waiters, and the waiters must not all retry.
func TestWaitersRetryWhenLeaderAuthorizeFails(t *testing.T) {
	var acquireCalls int32
	gate := newTokenGate()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token/" {
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		n := atomic.AddInt32(&acquireCalls, 1)
		gate.enter()
		if n == 1 {
			http.Error(w, "vms unavailable", http.StatusInternalServerError)
			return
		}
		writeTokenResponse(w, "access-"+strconv.Itoa(int(n)), "refresh-"+strconv.Itoa(int(n)))
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := newCoalesceTestAuth(host, port)

	const followers = 4
	results := make(chan error, followers+1)

	go func() { results <- auth.authorize() }()
	gate.waitReached()

	for i := 0; i < followers; i++ {
		go func() { results <- auth.authorize() }()
	}
	time.Sleep(followerParkTime)
	gate.releaseAll()

	failures := 0
	for i := 0; i < followers+1; i++ {
		if err := <-results; err != nil {
			failures++
		}
	}

	if failures != 1 {
		t.Errorf("failed authorize calls = %d, want 1 (only the leader saw the error)", failures)
	}
	if calls := atomic.LoadInt32(&acquireCalls); calls != 2 {
		t.Errorf("token acquisitions = %d, want 2 (one failure plus one retry)", calls)
	}
	assertTokenAndGeneration(t, auth, "access-2", 1)
}

// TestSequentialAuthorizeIsNotCoalesced guards the other direction: coalescing
// must not swallow deliberate sequential authorize calls, so each one still
// reaches the VMS.
func TestSequentialAuthorizeIsNotCoalesced(t *testing.T) {
	var refreshCalls, acquireCalls int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/token/":
			atomic.AddInt32(&acquireCalls, 1)
			writeTokenResponse(w, "acquired", "refresh-acquired")
		case "/api/token/refresh/":
			n := atomic.AddInt32(&refreshCalls, 1)
			writeTokenResponse(w, "refreshed-"+strconv.Itoa(int(n)), "refresh-"+strconv.Itoa(int(n)))
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := newCoalesceTestAuth(host, port)

	for i := 0; i < 3; i++ {
		if err := auth.authorize(); err != nil {
			t.Fatalf("sequential authorize %d failed: %v", i+1, err)
		}
	}

	if calls := atomic.LoadInt32(&acquireCalls); calls != 1 {
		t.Errorf("token acquisitions = %d, want 1", calls)
	}
	if calls := atomic.LoadInt32(&refreshCalls); calls != 2 {
		t.Errorf("token refreshes = %d, want 2", calls)
	}
	assertTokenAndGeneration(t, auth, "refreshed-2", 3)
}
