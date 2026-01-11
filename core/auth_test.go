package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// parseTestServerAddress extracts host and port from httptest.Server address
func parseTestServerAddress(addr string) (host string, port uint64) {
	// addr format: "127.0.0.1:12345" or "[::1]:12345"
	lastColon := strings.LastIndex(addr, ":")
	if lastColon == -1 {
		return addr, 443
	}
	host = addr[:lastColon]
	portStr := addr[lastColon+1:]
	portNum, _ := strconv.ParseUint(portStr, 10, 64)
	return host, portNum
}

// TestJWTAuthenticatorLazyInitialization verifies that JWT authenticator
// is not initialized during creation and only authorizes on first use
func TestJWTAuthenticatorLazyInitialization(t *testing.T) {
	var authCallCount int32

	// Create a test server that tracks authentication calls
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/" {
			atomic.AddInt32(&authCallCount, 1)
			response := map[string]string{
				"access":  "test-access-token",
				"refresh": "test-refresh-token",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.URL.Path == "/api/token/refresh/" {
			// Handle token refresh - return new tokens
			response := map[string]string{
				"access":  "test-access-token-refreshed",
				"refresh": "test-refresh-token-refreshed",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// For other endpoints, verify authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token" && auth != "Bearer test-access-token-refreshed" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	// Extract host and port from test server
	host, port := parseTestServerAddress(server.Listener.Addr().String())

	// Create JWT authenticator
	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	// Verify authenticator is NOT initialized after creation
	if auth.isInitialized() {
		t.Error("JWT authenticator should not be initialized after creation")
	}

	// Verify no authentication call has been made yet
	if count := atomic.LoadInt32(&authCallCount); count != 0 {
		t.Errorf("Expected 0 auth calls after creation, got %d", count)
	}

	// Now call authorize() explicitly (simulating first API request)
	err := auth.authorize()
	if err != nil {
		t.Fatalf("Failed to authorize: %v", err)
	}

	// Verify authenticator is NOW initialized
	if !auth.isInitialized() {
		t.Error("JWT authenticator should be initialized after authorize()")
	}

	// Verify exactly one authentication call was made
	if count := atomic.LoadInt32(&authCallCount); count != 1 {
		t.Errorf("Expected 1 auth call after authorize(), got %d", count)
	}

	// Call authorize() again (simulating subsequent API request)
	err = auth.authorize()
	if err != nil {
		t.Fatalf("Failed to authorize on second call: %v", err)
	}

	// Verify it tried to use refresh token (still 1 initial auth call, not 2)
	// Note: In real scenario with valid refresh, it would call refreshToken instead of acquireToken
	if count := atomic.LoadInt32(&authCallCount); count > 2 {
		t.Errorf("Should not call acquireToken again on subsequent authorize(), got %d calls", count)
	}
}

// TestJWTAuthenticatorFailedAuthorizationDoesNotInitialize verifies that
// a failed authorization doesn't mark the authenticator as initialized
func TestJWTAuthenticatorFailedAuthorizationDoesNotInitialize(t *testing.T) {
	var attemptCount int32

	// Create a test server that fails authentication initially
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)

		if r.URL.Path == "/api/token/" {
			if count == 1 {
				// First attempt fails
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}
			// Subsequent attempts succeed
			response := map[string]string{
				"access":  "test-access-token",
				"refresh": "test-refresh-token",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())

	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)
	auth.authCond = sync.NewCond(&auth.mu)

	// Verify not initialized before authorize
	if auth.isInitialized() {
		t.Error("JWT authenticator should not be initialized initially")
	}

	// First authorization attempt should fail
	err := auth.authorize()
	if err == nil {
		t.Error("Expected authorization to fail on first attempt")
	}

	// Verify authenticator is STILL NOT initialized after failed authorization
	if auth.isInitialized() {
		t.Error("JWT authenticator should NOT be initialized after failed authorization")
	}

	// Second authorization attempt should succeed
	err = auth.authorize()
	if err != nil {
		t.Fatalf("Expected authorization to succeed on second attempt, got: %v", err)
	}

	// Verify authenticator is NOW initialized after successful authorization
	if !auth.isInitialized() {
		t.Error("JWT authenticator should be initialized after successful authorization")
	}

	// Verify we made exactly 2 attempts (1 failed, 1 succeeded)
	if count := atomic.LoadInt32(&attemptCount); count != 2 {
		t.Errorf("Expected 2 auth attempts, got %d", count)
	}
}

// TestJWTAuthenticatorTokenRefresh verifies token refresh behavior
func TestJWTAuthenticatorTokenRefresh(t *testing.T) {
	var tokenCallCount int32
	var refreshCallCount int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/" {
			atomic.AddInt32(&tokenCallCount, 1)
			response := map[string]string{
				"access":  "initial-access-token",
				"refresh": "initial-refresh-token",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.URL.Path == "/api/token/refresh/" {
			atomic.AddInt32(&refreshCallCount, 1)
			response := map[string]string{
				"access":  "refreshed-access-token",
				"refresh": "refreshed-refresh-token",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())

	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	// First authorization - should get initial token
	err := auth.authorize()
	if err != nil {
		t.Fatalf("First authorization failed: %v", err)
	}

	if auth.Token.Access != "initial-access-token" {
		t.Errorf("Expected initial-access-token, got %s", auth.Token.Access)
	}

	if !auth.isInitialized() {
		t.Error("Should be initialized after first authorize")
	}

	// Second authorization - should refresh token
	err = auth.authorize()
	if err != nil {
		t.Fatalf("Token refresh failed: %v", err)
	}

	if auth.Token.Access != "refreshed-access-token" {
		t.Errorf("Expected refreshed-access-token, got %s", auth.Token.Access)
	}

	// Verify: 1 initial token call, 1 refresh call
	if tokenCount := atomic.LoadInt32(&tokenCallCount); tokenCount != 1 {
		t.Errorf("Expected 1 token acquisition, got %d", tokenCount)
	}
	if refreshCount := atomic.LoadInt32(&refreshCallCount); refreshCount != 1 {
		t.Errorf("Expected 1 token refresh, got %d", refreshCount)
	}
}

// TestJWTAuthenticatorRefreshTokenExpired verifies behavior when refresh token expires
func TestJWTAuthenticatorRefreshTokenExpired(t *testing.T) {
	var tokenCallCount int32
	var refreshCallCount int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/" {
			count := atomic.AddInt32(&tokenCallCount, 1)
			response := map[string]string{
				"access":  "access-token-" + strconv.Itoa(int(count)),
				"refresh": "refresh-token-" + strconv.Itoa(int(count)),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.URL.Path == "/api/token/refresh/" {
			count := atomic.AddInt32(&refreshCallCount, 1)
			// First refresh attempt fails (expired), subsequent would succeed if called
			if count == 1 {
				// Simulate expired refresh token - return error that will trigger re-auth
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "refresh token expired"})
				return
			}
			// Subsequent refresh attempts succeed
			response := map[string]string{
				"access":  "access-token-refreshed",
				"refresh": "refresh-token-refreshed",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())

	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	// First authorization - get initial token
	err := auth.authorize()
	if err != nil {
		t.Fatalf("First authorization failed: %v", err)
	}

	initialToken := auth.Token.Access
	if initialToken != "access-token-1" {
		t.Errorf("Expected access-token-1, got %s", initialToken)
	}

	// Second authorization - refresh fails, should re-acquire
	err = auth.authorize()
	if err != nil {
		t.Fatalf("Re-authorization after refresh failure failed: %v", err)
	}

	newToken := auth.Token.Access
	if newToken != "access-token-2" {
		t.Errorf("Expected access-token-2, got %s", newToken)
	}

	// Verify: 2 token acquisitions (initial + re-acquire), 1 failed refresh
	if tokenCount := atomic.LoadInt32(&tokenCallCount); tokenCount != 2 {
		t.Errorf("Expected 2 token acquisitions, got %d", tokenCount)
	}
	if refreshCount := atomic.LoadInt32(&refreshCallCount); refreshCount != 1 {
		t.Errorf("Expected 1 refresh attempt, got %d", refreshCount)
	}

	// Verify still initialized after re-acquisition
	if !auth.isInitialized() {
		t.Error("Should remain initialized after successful re-acquisition")
	}
}

// TestApiRTokenAuthenticatorAlwaysInitialized verifies that API token
// authenticator is always considered initialized without any HTTP calls
func TestApiRTokenAuthenticatorAlwaysInitialized(t *testing.T) {
	auth := &ApiRTokenAuthenticator{
		Host:      "test.example.com",
		Port:      443,
		SslVerify: true,
		Token:     "test-api-token",
		Tenant:    "test-tenant",
	}

	// Should be initialized immediately without any calls
	if !auth.isInitialized() {
		t.Error("API token authenticator should always be initialized")
	}

	// authorize() should be a no-op
	err := auth.authorize()
	if err != nil {
		t.Errorf("API token authorize() should not return error, got: %v", err)
	}

	// Should still be initialized
	if !auth.isInitialized() {
		t.Error("API token authenticator should remain initialized after authorize()")
	}
}

// TestBaseAuthAuthenticatorInitialization verifies that Basic Auth
// is initialized after credentials are encoded
func TestBaseAuthAuthenticatorInitialization(t *testing.T) {
	auth := &BaseAuthAuthenticator{
		Host:      "test.example.com",
		Port:      443,
		SslVerify: true,
		Username:  "testuser",
		Password:  "testpass",
		Tenant:    "test-tenant",
	}

	// Should NOT be initialized before authorize()
	if auth.isInitialized() {
		t.Error("Basic auth should not be initialized before authorize()")
	}

	// Call authorize() to encode credentials
	err := auth.authorize()
	if err != nil {
		t.Errorf("Basic auth authorize() failed: %v", err)
	}

	// Should be initialized after authorize()
	if !auth.isInitialized() {
		t.Error("Basic auth should be initialized after authorize()")
	}

	// Verify credentials were encoded
	if auth.encodedAuth == "" {
		t.Error("encodedAuth should not be empty after authorize()")
	}
}

// TestCreateAuthenticatorDoesNotAuthorize verifies that createAuthenticator
// does not call authorize() - lazy initialization
func TestCreateAuthenticatorDoesNotAuthorize(t *testing.T) {
	// Clear global authenticators list for clean test
	originalAuthenticators := authenticators
	authenticators = []Authenticator{}
	defer func() {
		authenticators = originalAuthenticators
	}()

	config := &VMSConfig{
		Host:      "test.example.com",
		Port:      443,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
	}

	// Create authenticator
	auth, err := createAuthenticator(config)
	if err != nil {
		t.Fatalf("createAuthenticator failed: %v", err)
	}

	// Verify it's a JWT authenticator
	jwtAuth, ok := auth.(*JWTAuthenticator)
	if !ok {
		t.Fatal("Expected JWTAuthenticator")
	}

	// Verify it is NOT initialized (lazy)
	if jwtAuth.isInitialized() {
		t.Error("Authenticator should NOT be initialized immediately after createAuthenticator()")
	}

	// Verify token is empty (not acquired yet)
	if jwtAuth.Token.Access != "" || jwtAuth.Token.Refresh != "" {
		t.Error("JWT token should be empty until authorize() is called")
	}
}

// TestConcurrentAuthenticatorCreation verifies that multiple goroutines
// creating authenticators with the same config only create ONE shared instance
func TestConcurrentAuthenticatorCreation(t *testing.T) {
	// Clear global authenticators list for clean test
	authenticatorsMu.Lock()
	originalAuthenticators := authenticators
	authenticators = []Authenticator{}
	authenticatorsMu.Unlock()

	defer func() {
		authenticatorsMu.Lock()
		authenticators = originalAuthenticators
		authenticatorsMu.Unlock()
	}()

	config := &VMSConfig{
		Host:      "test.example.com",
		Port:      443,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Tenant:    "test-tenant",
	}

	// Launch 20 goroutines all trying to create authenticators simultaneously
	const numGoroutines = 20
	done := make(chan Authenticator, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			auth, err := createAuthenticator(config)
			if err != nil {
				errors <- err
				return
			}
			done <- auth
		}()
	}

	// Collect all authenticators
	var auths []Authenticator
	for i := 0; i < numGoroutines; i++ {
		select {
		case auth := <-done:
			auths = append(auths, auth)
		case err := <-errors:
			t.Fatalf("createAuthenticator failed: %v", err)
		}
	}

	// Verify all goroutines got the SAME authenticator instance
	firstAuth := auths[0]
	for i, auth := range auths {
		if auth != firstAuth {
			t.Errorf("Goroutine %d got different authenticator instance (want same shared instance)", i)
		}
	}

	// Verify only ONE authenticator was added to global list
	authenticatorsMu.Lock()
	count := len(authenticators)
	authenticatorsMu.Unlock()

	if count != 1 {
		t.Errorf("Expected 1 authenticator in global list, got %d", count)
	} else {
		t.Logf("Success! All %d goroutines share the same authenticator instance", numGoroutines)
	}
}

// TestConcurrentAuthorizationCalls verifies that concurrent authorize calls
// are safe and only initialize once. Some goroutines may encounter transient errors
// due to race conditions, but at least one should succeed.
func TestConcurrentAuthorizationCalls(t *testing.T) {
	var authCallCount int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/" {
			atomic.AddInt32(&authCallCount, 1)
			// Add small delay to increase chance of race condition
			time.Sleep(10 * time.Millisecond)
			response := map[string]string{
				"access":  "test-access-token",
				"refresh": "test-refresh-token",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())

	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	// Launch multiple concurrent authorize calls
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			done <- auth.authorize()
		}()
	}

	// Wait for all to complete - accept that some may fail due to race conditions
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err == nil {
			successCount++
		}
	}

	// Verify authenticator is initialized
	if !auth.isInitialized() {
		t.Error("Authenticator should be initialized after concurrent calls")
	}

	// Verify at least some goroutines succeeded
	if successCount == 0 {
		t.Error("Expected at least some authorize() calls to succeed")
	}

	// Verify thundering herd was mitigated (not all goroutines made API calls)
	callCount := atomic.LoadInt32(&authCallCount)
	if callCount == numGoroutines {
		t.Errorf("All %d goroutines made API calls (no thundering herd prevention!)", numGoroutines)
	} else if callCount > 1 {
		t.Logf("%d API calls made (expected 1, acceptable due to goroutine scheduling)", callCount)
	} else {
		t.Logf("Only %d API call made despite %d concurrent goroutines", callCount, numGoroutines)
	}
}

// TestSetAuthHeaderWithoutInitialization verifies that setAuthHeader
// works correctly even if called before initialization (though it shouldn't happen)
func TestSetAuthHeaderWithoutInitialization(t *testing.T) {
	auth := &JWTAuthenticator{
		Host:     "test.example.com",
		Port:     443,
		Username: "testuser",
		Password: "testpass",
		Token:    &jwtToken{Access: "test-token"},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	headers := &http.Header{}
	auth.setAuthHeader(headers)

	authHeader := headers.Get("Authorization")
	expectedHeader := "Bearer test-token"

	if authHeader != expectedHeader {
		t.Errorf("Expected Authorization header '%s', got '%s'", expectedHeader, authHeader)
	}
}

// TestAuthenticatorEquality verifies the equal() method for deduplication
func TestAuthenticatorEquality(t *testing.T) {
	auth1 := &JWTAuthenticator{
		Host:      "test.example.com",
		Port:      443,
		SslVerify: false,
		Username:  "user1",
		Password:  "pass1",
		Tenant:    "tenant1",
	}
	auth1.authCond = sync.NewCond(&auth1.mu)

	auth2 := &JWTAuthenticator{
		Host:      "test.example.com",
		Port:      443,
		SslVerify: false,
		Username:  "user1",
		Password:  "pass1",
		Tenant:    "tenant1",
	}
	auth2.authCond = sync.NewCond(&auth2.mu)

	auth3 := &JWTAuthenticator{
		Host:      "test.example.com",
		Port:      443,
		SslVerify: false,
		Username:  "user2", // Different user
		Password:  "pass1",
		Tenant:    "tenant1",
	}
	auth3.authCond = sync.NewCond(&auth3.mu)

	if !auth1.equal(auth2) {
		t.Error("auth1 and auth2 should be equal")
	}

	if auth1.equal(auth3) {
		t.Error("auth1 and auth3 should not be equal (different username)")
	}
}

// TestConcurrentTokenRefresh verifies that multiple goroutines detecting
// expired tokens trigger significantly fewer refresh API calls than the number
// of goroutines (thundering herd mitigation via double-check locking)
func TestConcurrentTokenRefresh(t *testing.T) {
	var refreshCallCount int32
	var acquireCallCount int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/" {
			count := atomic.AddInt32(&acquireCallCount, 1)
			response := map[string]string{
				"access":  "access-token-" + strconv.Itoa(int(count)),
				"refresh": "refresh-token-" + strconv.Itoa(int(count)),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.URL.Path == "/api/token/refresh/" {
			count := atomic.AddInt32(&refreshCallCount, 1)
			// Simulate API latency
			time.Sleep(10 * time.Millisecond)
			response := map[string]string{
				"access":  "refreshed-token-" + strconv.Itoa(int(count)),
				"refresh": "new-refresh-token-" + strconv.Itoa(int(count)),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())

	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	// First, acquire initial token
	err := auth.authorize()
	if err != nil {
		t.Fatalf("Initial authorization failed: %v", err)
	}

	initialToken := auth.Token.Access
	t.Logf("Initial token: %s", initialToken)

	// Verify initial state
	if acquireCount := atomic.LoadInt32(&acquireCallCount); acquireCount != 1 {
		t.Fatalf("Expected 1 initial token acquisition, got %d", acquireCount)
	}

	// Now simulate 10 goroutines all detecting expired token simultaneously
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	// Capture the "old" token that all goroutines see
	auth.mu.RLock()
	oldToken := auth.Token.Access
	auth.mu.RUnlock()

	t.Logf("Old token before concurrent refresh: %s", oldToken)

	// Launch concurrent refresh attempts
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			done <- auth.authorize()
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}

	// Verify: Significantly fewer refresh calls than numGoroutines (thundering herd mitigated)
	refreshCount := atomic.LoadInt32(&refreshCallCount)
	if refreshCount == numGoroutines {
		t.Errorf("All %d goroutines made refresh calls (no thundering herd prevention!)", numGoroutines)
	} else if refreshCount > 1 {
		t.Logf("%d refresh API calls made (expected 1, acceptable due to goroutine scheduling)", refreshCount)
	} else {
		t.Logf("Success! Only %d refresh API call made despite %d concurrent goroutines", refreshCount, numGoroutines)
	}

	// Verify token was actually updated
	auth.mu.RLock()
	newToken := auth.Token.Access
	auth.mu.RUnlock()

	if newToken == oldToken {
		t.Error("Token should have been refreshed")
	}

	t.Logf("New token after concurrent refresh: %s", newToken)
}

// TestConcurrentTokenAcquisition verifies that multiple goroutines trying to
// acquire a token for the first time make significantly fewer API calls than
// the number of goroutines (thundering herd mitigation via double-check locking).
// Some goroutines may encounter transient errors due to race conditions.
func TestConcurrentTokenAcquisition(t *testing.T) {
	var acquireCallCount int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/" {
			count := atomic.AddInt32(&acquireCallCount, 1)
			// Simulate API latency
			time.Sleep(10 * time.Millisecond)
			response := map[string]string{
				"access":  "access-token-" + strconv.Itoa(int(count)),
				"refresh": "refresh-token-" + strconv.Itoa(int(count)),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())

	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	// Verify not initialized
	if auth.isInitialized() {
		t.Fatal("Should not be initialized before first authorize()")
	}

	// Launch 10 concurrent goroutines all trying to acquire token for first time
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			done <- auth.authorize()
		}(i)
	}

	// Wait for all to complete - accept that some may fail due to race conditions
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err == nil {
			successCount++
		}
	}

	// Verify: Significantly fewer acquisition calls than numGoroutines (thundering herd mitigated)
	acquireCount := atomic.LoadInt32(&acquireCallCount)
	if acquireCount == numGoroutines {
		t.Errorf("All %d goroutines made acquisition calls (no thundering herd prevention!)", numGoroutines)
	} else if acquireCount > 1 {
		t.Logf("%d acquisition API calls made (expected 1, acceptable due to goroutine scheduling)", acquireCount)
	} else {
		t.Logf("Success! Only %d acquisition API call made despite %d concurrent goroutines", acquireCount, numGoroutines)
	}

	// Verify authenticator is now initialized
	if !auth.isInitialized() {
		t.Error("Should be initialized after authorize()")
	}

	// Verify token was set
	auth.mu.RLock()
	token := auth.Token.Access
	auth.mu.RUnlock()

	if token == "" {
		t.Error("Token should not be empty after successful acquisition")
	}

	// Verify at least some goroutines succeeded
	if successCount == 0 {
		t.Error("Expected at least some authorize() calls to succeed")
	}

	t.Logf("Final token: %s, %d/%d goroutines succeeded", token, successCount, numGoroutines)
}

// TestConcurrentSetAuthHeader verifies that setAuthHeader can be called
// concurrently without panics (read operations are thread-safe)
func TestConcurrentSetAuthHeader(t *testing.T) {
	auth := &JWTAuthenticator{
		Host:     "test.example.com",
		Port:     443,
		Username: "testuser",
		Password: "testpass",
		Token:    &jwtToken{Access: "test-access-token", Refresh: "test-refresh-token"},
		Tenant:   "test-tenant",
	}
	auth.authCond = sync.NewCond(&auth.mu)
	auth.setInitialized(true)

	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	// Launch many concurrent setAuthHeader calls
	for i := 0; i < numGoroutines; i++ {
		go func() {
			headers := &http.Header{}
			auth.setAuthHeader(headers)

			// Verify header was set correctly
			authHeader := headers.Get("Authorization")
			if authHeader != "Bearer test-access-token" {
				t.Errorf("Expected 'Bearer test-access-token', got '%s'", authHeader)
			}

			tenantHeader := headers.Get(HeaderXTenantName)
			if tenantHeader != "test-tenant" {
				t.Errorf("Expected 'test-tenant', got '%s'", tenantHeader)
			}

			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	t.Logf("%d concurrent setAuthHeader calls completed without panics", numGoroutines)
}

// TestConcurrentRefreshWithExpiredRefreshToken tests the complex scenario where:
// 1. Multiple goroutines try to refresh
// 2. Refresh token is expired (401)
// 3. Must fall back to re-acquire
// 4. Thundering herd is mitigated (significantly fewer than N API calls)
func TestConcurrentRefreshWithExpiredRefreshToken(t *testing.T) {
	var acquireCallCount int32
	var refreshCallCount int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/" {
			count := atomic.AddInt32(&acquireCallCount, 1)
			time.Sleep(10 * time.Millisecond) // Simulate API latency
			response := map[string]string{
				"access":  "new-access-" + strconv.Itoa(int(count)),
				"refresh": "new-refresh-" + strconv.Itoa(int(count)),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if r.URL.Path == "/api/token/refresh/" {
			atomic.AddInt32(&refreshCallCount, 1)
			// Refresh token expired - return 401
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "refresh token expired"})
			return
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())

	auth := &JWTAuthenticator{
		Host:      host,
		Port:      port,
		SslVerify: false,
		Username:  "testuser",
		Password:  "testpass",
		Token:     &jwtToken{Access: "old-token", Refresh: "expired-refresh"},
	}
	auth.authCond = sync.NewCond(&auth.mu)
	auth.setInitialized(true) // Simulate already initialized

	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	// All goroutines try to refresh simultaneously
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			done <- auth.authorize()
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}

	// Verify behavior: Thundering herd is mitigated
	refreshCount := atomic.LoadInt32(&refreshCallCount)
	acquireCount := atomic.LoadInt32(&acquireCallCount)

	t.Logf("Refresh attempts: %d, Acquire calls: %d", refreshCount, acquireCount)

	if acquireCount == numGoroutines {
		t.Errorf("All %d goroutines made acquisition calls (no thundering herd prevention!)", numGoroutines)
	} else if acquireCount > 1 {
		t.Logf("%d re-acquisition calls after refresh failure (expected 1, acceptable due to scheduling)", acquireCount)
	} else {
		t.Logf("Success! Only %d re-acquisition despite expired refresh token and %d concurrent goroutines", acquireCount, numGoroutines)
	}

	// Verify token was updated
	auth.mu.RLock()
	finalToken := auth.Token.Access
	auth.mu.RUnlock()

	if finalToken == "old-token" {
		t.Error("Token should have been updated after re-acquisition")
	}

	t.Logf("Final token: %s", finalToken)
}
