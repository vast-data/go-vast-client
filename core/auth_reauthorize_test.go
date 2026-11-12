package core

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeVMS serves the JWT token endpoints and delegates everything else to
// resource. Logins return access-aN, refreshes return access-rN.
type fakeVMS struct {
	t           *testing.T
	acquires    int32
	refreshes   int32
	acquireGate *tokenGate        // optional: hold logins until released
	acquireFail func(n int32) int // optional: status to fail login N with, 0 = succeed
	refreshFail func(n int32) int // optional: status to fail refresh N with, 0 = succeed
	resource    func(w http.ResponseWriter, r *http.Request)
}

func (f *fakeVMS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/token/":
		n := atomic.AddInt32(&f.acquires, 1)
		if f.acquireGate != nil {
			f.acquireGate.enter()
		}
		if f.acquireFail != nil {
			if status := f.acquireFail(n); status != 0 {
				http.Error(w, `{"detail":"login failed"}`, status)
				return
			}
		}
		writeTokenResponse(w, "access-a"+strconv.Itoa(int(n)), "refresh-a"+strconv.Itoa(int(n)))
	case "/api/token/refresh/":
		n := atomic.AddInt32(&f.refreshes, 1)
		if f.refreshFail != nil {
			if status := f.refreshFail(n); status != 0 {
				http.Error(w, `{"detail":"refresh failed"}`, status)
				return
			}
		}
		writeTokenResponse(w, "access-r"+strconv.Itoa(int(n)), "refresh-r"+strconv.Itoa(int(n)))
	default:
		f.resource(w, r)
	}
}

func (f *fakeVMS) counts() (acquires, refreshes int32) {
	return atomic.LoadInt32(&f.acquires), atomic.LoadInt32(&f.refreshes)
}

func writeOK(w http.ResponseWriter) {
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	_, _ = w.Write([]byte(`{}`))
}

func bearer(token string) string {
	return AuthTypeBearer + " " + token
}

// newJWTTestSession builds a session with username/password auth. The username
// is unique per test because createAuthenticator shares equal authenticators.
func newJWTTestSession(t *testing.T, server *httptest.Server) *VMSSession {
	t.Helper()
	host, port := parseTestServerAddress(server.Listener.Addr().String())
	timeout := time.Minute
	session, err := NewVMSSession(&VMSConfig{
		Host:           host,
		Port:           port,
		Username:       "user-" + t.Name(),
		Password:       "pass",
		SslVerify:      false,
		Timeout:        &timeout,
		MaxConnections: 50,
		ApiVersion:     "latest",
	})
	if err != nil {
		t.Fatalf("NewVMSSession: %v", err)
	}
	return session
}

func jwtOf(session *VMSSession) *JWTAuthenticator { return session.auth.(*JWTAuthenticator) }

// ageToken makes the current token older than the reuse window; noWindow
// disables reuse so each rejection renews deterministically.
func ageToken(session *VMSSession) {
	j := jwtOf(session)
	j.mu.Lock()
	j.lastAuthorized = time.Now().Add(-time.Hour)
	j.mu.Unlock()
}

func noWindow(session *VMSSession) { jwtOf(session).reuseWindow = 0 }

func getViews(session *VMSSession) error {
	_, err := session.Get(context.Background(), "views", nil, nil)
	return err
}

func runConcurrently(n int, fn func() error) []error {
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = fn()
		}(i)
	}
	wg.Wait()
	return errs
}

func requireNoErrors(t *testing.T, errs []error) {
	t.Helper()
	for i, err := range errs {
		if err != nil {
			t.Errorf("request %d failed: %v", i, err)
		}
	}
}

func requireCounts(t *testing.T, vms *fakeVMS, wantAcquires, wantRefreshes int32) {
	t.Helper()
	acquires, refreshes := vms.counts()
	if acquires != wantAcquires {
		t.Errorf("logins = %d, want %d", acquires, wantAcquires)
	}
	if refreshes != wantRefreshes {
		t.Errorf("refreshes = %d, want %d", refreshes, wantRefreshes)
	}
}

// ---------------------------------------------------------------------------
// Empty initial token: lazy login in setupHeaders
// ---------------------------------------------------------------------------

// Many first requests while the leader's login is in flight share one login.
func TestEmptyToken_ConcurrentFirstRequestsLogInOnce(t *testing.T) {
	gate := newTokenGate()
	vms := &fakeVMS{t: t, acquireGate: gate}
	vms.resource = func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(HeaderAuthorization); got != bearer("access-a1") {
			t.Errorf("request sent with %q, want the leader's token", got)
		}
		writeOK(w)
	}
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	session := newJWTTestSession(t, server)

	const requests = 20
	errs := make(chan error, requests)
	go func() { errs <- getViews(session) }()
	gate.waitReached()
	for i := 1; i < requests; i++ {
		go func() { errs <- getViews(session) }()
	}
	time.Sleep(followerParkTime)
	gate.releaseAll()

	for i := 0; i < requests; i++ {
		if err := <-errs; err != nil {
			t.Errorf("request failed: %v", err)
		}
	}
	requireCounts(t, vms, 1, 0)
}

// When the first login fails while others wait, exactly one waiter takes over
// and the rest reuse its token.
func TestEmptyToken_WaitersTakeOverAfterFailedLogin(t *testing.T) {
	gate := newTokenGate()
	vms := &fakeVMS{t: t, acquireGate: gate, acquireFail: func(n int32) int {
		if n == 1 {
			return http.StatusServiceUnavailable
		}
		return 0
	}}
	vms.resource = func(w http.ResponseWriter, r *http.Request) { writeOK(w) }
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	session := newJWTTestSession(t, server)

	const requests = 5
	errs := make(chan error, requests)
	go func() { errs <- getViews(session) }()
	gate.waitReached()
	for i := 1; i < requests; i++ {
		go func() { errs <- getViews(session) }()
	}
	time.Sleep(followerParkTime)
	gate.releaseAll()

	failures := 0
	for i := 0; i < requests; i++ {
		if err := <-errs; err != nil {
			failures++
		}
	}
	if failures != 1 {
		t.Errorf("failed requests = %d, want 1 (only the leader)", failures)
	}
	requireCounts(t, vms, 2, 0)
}

// A failed first login leaves no token; the next request logs in again.
func TestEmptyToken_FailedFirstLoginIsRetriedByNextRequest(t *testing.T) {
	vms := &fakeVMS{t: t, acquireFail: func(n int32) int {
		if n == 1 {
			return http.StatusServiceUnavailable
		}
		return 0
	}}
	vms.resource = func(w http.ResponseWriter, r *http.Request) { writeOK(w) }
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	session := newJWTTestSession(t, server)

	if err := getViews(session); err == nil {
		t.Fatal("first request succeeded, want login error")
	}
	if err := getViews(session); err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	requireCounts(t, vms, 2, 0)
}

// ---------------------------------------------------------------------------
// 401/403 rejection: reauthorize in doRequestWithRetries
// ---------------------------------------------------------------------------

var rejectionStatuses = []int{http.StatusUnauthorized, http.StatusForbidden}

// A rejection that arrives after a peer already renewed the token reuses the
// renewed token instead of refreshing again.
func TestRejected_LateResponseReusesRenewedToken(t *testing.T) {
	for _, status := range rejectionStatuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var staleCalls int32
			bothSent := make(chan struct{})
			firstRetried := make(chan struct{})
			var firstRetriedOnce sync.Once

			vms := &fakeVMS{t: t}
			vms.resource = func(w http.ResponseWriter, r *http.Request) {
				switch r.Header.Get(HeaderAuthorization) {
				case bearer("access-a1"):
					if atomic.AddInt32(&staleCalls, 1) == 1 {
						<-bothSent
					} else {
						close(bothSent)
						select {
						case <-firstRetried:
						case <-time.After(10 * time.Second):
							t.Error("first request never retried with the renewed token")
						}
					}
					http.Error(w, `{"detail":"token expired"}`, status)
				case bearer("access-r1"):
					firstRetriedOnce.Do(func() { close(firstRetried) })
					writeOK(w)
				default:
					t.Errorf("unexpected authorization %q", r.Header.Get(HeaderAuthorization))
					http.Error(w, "unexpected", http.StatusBadRequest)
				}
			}
			server := httptest.NewTLSServer(vms)
			defer server.Close()
			session := newJWTTestSession(t, server)
			if err := session.auth.authorize(); err != nil {
				t.Fatalf("initial login failed: %v", err)
			}
			ageToken(session)

			requireNoErrors(t, runConcurrently(2, func() error { return getViews(session) }))
			requireCounts(t, vms, 1, 1)
		})
	}
}

// Many requests rejected at the same moment share one refresh.
func TestRejected_ConcurrentResponsesRefreshOnce(t *testing.T) {
	for _, status := range rejectionStatuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			const requests = 20
			var staleCalls int32
			allSent := make(chan struct{})

			vms := &fakeVMS{t: t}
			vms.resource = func(w http.ResponseWriter, r *http.Request) {
				switch r.Header.Get(HeaderAuthorization) {
				case bearer("access-a1"):
					if atomic.AddInt32(&staleCalls, 1) == requests {
						close(allSent)
					}
					select {
					case <-allSent:
					case <-time.After(10 * time.Second):
						t.Error("not all requests were sent with the old token")
					}
					http.Error(w, `{"detail":"token expired"}`, status)
				case bearer("access-r1"):
					writeOK(w)
				default:
					t.Errorf("unexpected authorization %q", r.Header.Get(HeaderAuthorization))
					http.Error(w, "unexpected", http.StatusBadRequest)
				}
			}
			server := httptest.NewTLSServer(vms)
			defer server.Close()
			session := newJWTTestSession(t, server)
			if err := session.auth.authorize(); err != nil {
				t.Fatalf("initial login failed: %v", err)
			}
			ageToken(session)

			requireNoErrors(t, runConcurrently(requests, func() error { return getViews(session) }))
			requireCounts(t, vms, 1, 1)
		})
	}
}

// A 403 for missing permissions is not an authentication problem: no renewal,
// and the error is returned without retrying.
func TestRejected_PermissionDeniedDoesNotReauthorize(t *testing.T) {
	var resourceCalls int32
	vms := &fakeVMS{t: t}
	vms.resource = func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&resourceCalls, 1)
		http.Error(w, `{"detail":"permission_denied"}`, http.StatusForbidden)
	}
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	session := newJWTTestSession(t, server)

	if err := getViews(session); err == nil {
		t.Fatal("request succeeded, want permission error")
	}
	if calls := atomic.LoadInt32(&resourceCalls); calls != 1 {
		t.Errorf("resource calls = %d, want 1 (no retry)", calls)
	}
	requireCounts(t, vms, 1, 0)
}

// A token the VMS keeps rejecting stops after maxRetries attempts, renewing
// before each retry, and returns the rejection.
func TestRejected_PersistentRejectionIsBounded(t *testing.T) {
	var resourceCalls int32
	vms := &fakeVMS{t: t}
	vms.resource = func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&resourceCalls, 1)
		http.Error(w, `{"detail":"token expired"}`, http.StatusUnauthorized)
	}
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	session := newJWTTestSession(t, server)
	noWindow(session)

	err := getViews(session)
	var apiErr *ApiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("error = %v, want ApiError 401", err)
	}
	if calls := atomic.LoadInt32(&resourceCalls); calls != int32(maxRetries) {
		t.Errorf("resource calls = %d, want %d", calls, maxRetries)
	}
	requireCounts(t, vms, 1, int32(maxRetries))
}

// An expired refresh token falls back to a full login.
func TestRejected_ExpiredRefreshTokenFallsBackToLogin(t *testing.T) {
	vms := &fakeVMS{t: t, refreshFail: func(int32) int { return http.StatusUnauthorized }}
	vms.resource = func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(HeaderAuthorization) == bearer("access-a2") {
			writeOK(w)
			return
		}
		http.Error(w, `{"detail":"token expired"}`, http.StatusUnauthorized)
	}
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	session := newJWTTestSession(t, server)
	noWindow(session)

	if err := getViews(session); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	requireCounts(t, vms, 2, 1)
}

// A failed renewal is returned to the caller and leaves no token, so the next
// request renews again instead of reusing the rejected one.
func TestRejected_FailedRenewalIsRetriedByNextRequest(t *testing.T) {
	vms := &fakeVMS{t: t, refreshFail: func(n int32) int {
		if n == 1 {
			return http.StatusServiceUnavailable
		}
		return 0
	}}
	vms.resource = func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get(HeaderAuthorization) {
		case bearer("access-a1"):
			http.Error(w, `{"detail":"token expired"}`, http.StatusUnauthorized)
		case bearer("access-r2"):
			writeOK(w)
		default:
			t.Errorf("unexpected authorization %q", r.Header.Get(HeaderAuthorization))
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	session := newJWTTestSession(t, server)
	noWindow(session)

	if err := getViews(session); err == nil {
		t.Fatal("first request succeeded, want renewal error")
	}
	if err := getViews(session); err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	requireCounts(t, vms, 1, 2)
}

// ---------------------------------------------------------------------------
// Authenticator level
// ---------------------------------------------------------------------------

// authorize reuses a token installed within the window, renews once it is
// older, and never reuses when the window is disabled.
func TestAuthorizeReuseWindow(t *testing.T) {
	vms := &fakeVMS{t: t, resource: func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s", r.URL.Path)
	}}
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := newCoalesceTestAuth(host, port)
	auth.reuseWindow = time.Hour

	for i := 0; i < 3; i++ { // first logs in, the rest fall inside the window
		if err := auth.authorize(); err != nil {
			t.Fatalf("authorize %d failed: %v", i+1, err)
		}
	}
	requireCounts(t, vms, 1, 0)
	assertToken(t, auth, "access-a1")

	auth.lastAuthorized = time.Now().Add(-2 * time.Hour) // token now older than window
	if err := auth.authorize(); err != nil {
		t.Fatalf("authorize after window failed: %v", err)
	}
	requireCounts(t, vms, 1, 1)
	assertToken(t, auth, "access-r1")

	auth.reuseWindow = 0 // disabled: always renews
	if err := auth.authorize(); err != nil {
		t.Fatalf("authorize with window disabled failed: %v", err)
	}
	requireCounts(t, vms, 1, 2)
}

// A failed authorize must not open the window: the next call retries.
func TestAuthorizeReuseWindowNotOpenedByFailure(t *testing.T) {
	vms := &fakeVMS{t: t, acquireFail: func(n int32) int {
		if n == 1 {
			return http.StatusServiceUnavailable
		}
		return 0
	}, resource: func(w http.ResponseWriter, r *http.Request) {}}
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := newCoalesceTestAuth(host, port)
	auth.reuseWindow = time.Hour

	if err := auth.authorize(); err == nil {
		t.Fatal("first authorize succeeded, want error")
	}
	if err := auth.authorize(); err != nil {
		t.Fatalf("second authorize failed: %v", err)
	}
	requireCounts(t, vms, 2, 0)
}

// Waiters rejected with the same token: when the leader's refresh fails, one
// waiter takes over and the rest reuse its token.
func TestAuthorizeIfStale_WaitersTakeOverAfterFailedRefresh(t *testing.T) {
	gate := newTokenGate()
	var refreshes int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&refreshes, 1)
		gate.enter()
		if n == 1 {
			http.Error(w, "vms unavailable", http.StatusServiceUnavailable)
			return
		}
		writeTokenResponse(w, "access-r"+strconv.Itoa(int(n)), "refresh-r"+strconv.Itoa(int(n)))
	}))
	defer server.Close()
	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := newCoalesceTestAuth(host, port)
	auth.Token = &jwtToken{Access: "stale-token", Refresh: "stale-refresh"}
	auth.setInitialized(true)

	const waiters = 5
	results := make(chan error, waiters)
	go func() { results <- auth.authorize() }()
	gate.waitReached()
	for i := 1; i < waiters; i++ {
		go func() { results <- auth.authorize() }()
	}
	time.Sleep(followerParkTime)
	gate.releaseAll()

	failures := 0
	for i := 0; i < waiters; i++ {
		if err := <-results; err != nil {
			failures++
		}
	}
	if failures != 1 {
		t.Errorf("failed calls = %d, want 1", failures)
	}
	if got := atomic.LoadInt32(&refreshes); got != 2 {
		t.Errorf("refreshes = %d, want 2 (failure plus one takeover)", got)
	}
	assertToken(t, auth, "access-r2")
}

// API-token and Basic auth have no renewal: a rejection retries (bounded) via
// plain authorize and never touches token endpoints.
func TestRejected_StaticCredentialsRetryBounded(t *testing.T) {
	for _, name := range []string{"api-token", "basic"} {
		t.Run(name, func(t *testing.T) {
			var calls int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/token") {
					t.Errorf("unexpected token request %s", r.URL.Path)
				}
				atomic.AddInt32(&calls, 1)
				http.Error(w, `{"detail":"bad credentials"}`, http.StatusUnauthorized)
			}))
			defer server.Close()
			host, port := parseTestServerAddress(server.Listener.Addr().String())
			timeout := time.Minute
			cfg := &VMSConfig{Host: host, Port: port, SslVerify: false, Timeout: &timeout,
				MaxConnections: 5, ApiVersion: "latest"}
			if name == "api-token" {
				cfg.ApiToken = "static-" + t.Name()
			} else {
				cfg.UseBasicAuth, cfg.Username, cfg.Password = true, "basic-"+t.Name(), "pw"
			}
			session, err := NewVMSSession(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if err := getViews(session); err == nil {
				t.Fatal("request succeeded, want 401")
			}
			if got := atomic.LoadInt32(&calls); got != int32(maxRetries) {
				t.Errorf("requests = %d, want %d", got, maxRetries)
			}
		})
	}
}

// A rejection that arrives seconds after a peer renewed (a slow request) is
// still covered by the default window, but not once it has passed.
func TestDefaultReuseWindowCoversSlowRejections(t *testing.T) {
	vms := &fakeVMS{t: t, resource: func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s", r.URL.Path)
	}}
	server := httptest.NewTLSServer(vms)
	defer server.Close()
	session := newJWTTestSession(t, server)
	auth := jwtOf(session)

	if auth.reuseWindow < 25*time.Second {
		t.Fatalf("default reuse window = %v, want it to outlast slow requests (>= 25s)", auth.reuseWindow)
	}
	if err := auth.authorize(); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	auth.mu.Lock()
	auth.lastAuthorized = time.Now().Add(-10 * time.Second) // peer renewed 10s ago
	auth.mu.Unlock()
	if err := auth.authorize(); err != nil {
		t.Fatalf("late authorize failed: %v", err)
	}
	requireCounts(t, vms, 1, 0)

	auth.mu.Lock()
	auth.lastAuthorized = time.Now().Add(-auth.reuseWindow - time.Second)
	auth.mu.Unlock()
	if err := auth.authorize(); err != nil {
		t.Fatalf("authorize after window failed: %v", err)
	}
	requireCounts(t, vms, 1, 1)
}
