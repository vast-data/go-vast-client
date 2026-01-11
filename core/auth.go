package core

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

var (
	authenticators   []Authenticator
	authenticatorsMu sync.Mutex
)

type Authenticator interface {
	authorize() error
	setAuthHeader(headers *http.Header)
	equal(other Authenticator) bool
	setInitialized(bool)
	isInitialized() bool
}

// createAuthenticator creates a new Authenticator instance based on the provided VMSConfig.
// Each session gets its own authenticator instance to avoid global state issues.
func createAuthenticator(config *VMSConfig) (Authenticator, error) {
	var authenticator Authenticator

	// Priority: ApiToken > BasicAuth > JWT
	if config.ApiToken != "" {
		authenticator = &ApiRTokenAuthenticator{
			Host:      config.Host,
			Port:      config.Port,
			SslVerify: config.SslVerify,
			Token:     config.ApiToken,
			Tenant:    config.Tenant,
		}
	} else if config.UseBasicAuth && config.Username != "" && config.Password != "" {
		authenticator = &BaseAuthAuthenticator{
			Host:      config.Host,
			Port:      config.Port,
			SslVerify: config.SslVerify,
			Username:  config.Username,
			Password:  config.Password,
			Tenant:    config.Tenant,
		}
	} else if config.Username != "" && config.Password != "" {
		jwtAuth := &JWTAuthenticator{
			Host:         config.Host,
			Port:         config.Port,
			SslVerify:    config.SslVerify,
			RespectProxy: config.RespectProxy,
			Username:     config.Username,
			Password:     config.Password,
			Tenant:       config.Tenant,
			Token:        &jwtToken{},
		}
		jwtAuth.authCond = sync.NewCond(&jwtAuth.mu)
		authenticator = jwtAuth
	}
	if authenticator != nil {
		authenticatorsMu.Lock()
		defer authenticatorsMu.Unlock()

		for _, existingAuthenticator := range authenticators {
			if existingAuthenticator.equal(authenticator) {
				return existingAuthenticator, nil
			}
		}
		authenticators = append(authenticators, authenticator)
		return authenticator, nil
	}

	panic("CreateAuthenticator: neither username/password nor apiToken are provided")
}

type jwtToken struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type JWTAuthenticator struct {
	Host         string
	Port         uint64
	SslVerify    bool
	RespectProxy bool
	Username     string
	Password     string
	Token        *jwtToken
	Tenant       string
	initialized  bool
	mu           sync.RWMutex // Protects Token and initialized
	authorizing  bool         // Indicates authorization in progress
	authCond     *sync.Cond   // Condition variable for waiting goroutines
}

func parseToken(rsp *http.Response) (*jwtToken, error) {
	var tokens jwtToken
	out, e := io.ReadAll(rsp.Body)
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal(out, &tokens)
	if e != nil {
		return nil, e
	}
	return &tokens, nil
}

func (auth *JWTAuthenticator) refreshToken(client *http.Client) error {
	auth.mu.RLock()
	if auth.Token == nil || auth.Token.Refresh == "" {
		panic("refreshToken called without valid token - auth.initialized state is corrupted!")
	}
	refreshToken := auth.Token.Refresh
	auth.mu.RUnlock()

	server := auth.Host + ":" + strconv.FormatUint(auth.Port, 10)
	path := url.URL{
		Scheme: "https",
		Host:   server,
		Path:   "api/token/refresh/",
	}
	body, err := json.Marshal(map[string]string{"refresh": refreshToken})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, path.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set(HeaderContentType, ContentTypeJSON)
	if auth.Tenant != "" {
		req.Header.Set(HeaderXTenantName, auth.Tenant)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Validate response first (reads body only on error)
	if err := validateResponse(resp, auth.Host, auth.Port); err != nil {
		return err
	}

	// Parse the token (reads body on success path)
	token, parseErr := parseToken(resp)
	if parseErr != nil {
		return parseErr
	}

	auth.mu.Lock()
	auth.Token = token
	auth.mu.Unlock()

	return nil
}

func (auth *JWTAuthenticator) acquireToken(client *http.Client) error {
	userPass := map[string]string{"username": auth.Username, "password": auth.Password}
	server := auth.Host + ":" + strconv.FormatUint(auth.Port, 10)
	body, err := json.Marshal(userPass)
	if err != nil {
		return err
	}
	path := url.URL{
		Scheme: "https",
		Host:   server,
		Path:   "api/token/",
	}

	req, err := http.NewRequest(http.MethodPost, path.String(), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set(HeaderContentType, ContentTypeJSON)
	if auth.Tenant != "" {
		req.Header.Set(HeaderXTenantName, auth.Tenant)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Validate response first (reads body only on error)
	if err := validateResponse(resp, auth.Host, auth.Port); err != nil {
		return err
	}

	// Parse the token (reads body on success path)
	token, parseErr := parseToken(resp)
	if parseErr != nil {
		return parseErr
	}

	auth.mu.Lock()
	auth.Token = token
	auth.mu.Unlock()

	return nil
}

// authorize acquires or refreshes the JWT token for API authentication.
//
// This method implements a thread-safe, single-authorization-at-a-time pattern
// to prevent the "thundering herd" problem where multiple concurrent goroutines
// would all attempt to acquire/refresh tokens simultaneously, causing redundant
// API calls and potential rate limiting.
//
// Concurrency Strategy:
//
//  1. Authorization In Progress Flag (auth.authorizing):
//     - Acts as a signal that one goroutine is currently performing authorization
//     - Protected by auth.mu mutex for thread-safe access
//
//  2. Condition Variable (auth.authCond):
//     - Coordinates goroutines waiting for authorization to complete
//     - Wait() atomically: releases lock → sleeps → re-acquires lock when signaled
//     - Broadcast() wakes all waiting goroutines when authorization completes
//
//  3. Token Clearing Strategy:
//     - Before attempting authorization, we clear auth.Token.Access
//     - This ensures waiting goroutines won't use stale/invalid tokens if authorization fails
//     - If authorization succeeds, the new token is written; if it fails, token stays empty
//
// Flow for Concurrent Calls:
//
//	Goroutine 1 (first to arrive):
//	  → Acquires lock
//	  → Sets auth.authorizing = true
//	  → Clears auth.Token.Access (invalidate old token)
//	  → Releases lock
//	  → Makes HTTP call to acquire/refresh token
//	  → Sets auth.authorizing = false, Broadcast() to wake waiters
//
//	Goroutines 2-N (arrive while G1 is working):
//	  → Acquire lock
//	  → See auth.authorizing = true
//	  → Call authCond.Wait() - releases lock and sleeps
//	  → Woken by Broadcast() when G1 completes
//	  → Re-acquire lock and check if token is now available:
//	     - If token exists → return (use G1's token)
//	     - If token is empty → G2 tries authorization (G1 failed)
//
// This design ensures:
//   - Only 1 HTTP call per authorization attempt (no thundering herd)
//   - Automatic retry on transient failures (next waiting goroutine tries)
//   - Thread-safe access to shared auth.Token state
func (auth *JWTAuthenticator) authorize() error {
	// Acquire lock and check if authorization is already in progress
	auth.mu.Lock()

	// Wait while another goroutine is authorizing
	if auth.authorizing {
		for auth.authorizing {
			auth.authCond.Wait() // Releases lock and waits, re-acquires when signaled
		}
		// We were waiting - check if token is now available
		if auth.initialized && auth.Token != nil && auth.Token.Access != "" {
			auth.mu.Unlock()
			return nil // Another goroutine got the token while we waited
		}
	}

	// We're the first - set authorizing flag and capture state
	auth.authorizing = true
	isInitialized := auth.initialized

	// Clear the access token before attempting authorization
	// This ensures waiting goroutines won't use stale token if authorization fails
	if auth.Token != nil {
		auth.Token.Access = ""
	}

	auth.mu.Unlock() // Release lock before making HTTP calls

	// Now make HTTP calls without holding the lock
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !auth.SslVerify},
	}
	if auth.RespectProxy {
		tr.Proxy = http.ProxyFromEnvironment
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   20 * time.Second,
	}

	var err error
	if isInitialized {
		err = auth.refreshToken(client)
		// If there is an error while getting new token using refresh token and
		// that error is API error with status code 401, then refresh token is also
		// expired. Need to re-authenticate.
		if err != nil && IsApiError(err) {
			statusCode := err.(*ApiError).StatusCode
			if statusCode == http.StatusUnauthorized {
				err = auth.acquireToken(client)
			}
		}
	} else {
		err = auth.acquireToken(client)
	}

	// Clear authorizing flag and notify waiting goroutines
	auth.mu.Lock()
	if err == nil {
		auth.initialized = true
	}
	auth.authorizing = false
	auth.authCond.Broadcast() // Wake up all waiting goroutines
	auth.mu.Unlock()

	return err
}

func (auth *JWTAuthenticator) setAuthHeader(headers *http.Header) {
	auth.mu.RLock()
	defer auth.mu.RUnlock()

	headers.Add(HeaderAuthorization, AuthTypeBearer+" "+auth.Token.Access)
	if auth.Tenant != "" {
		headers.Add(HeaderXTenantName, auth.Tenant)
	}
}

func (auth *JWTAuthenticator) equal(other Authenticator) bool {
	otherAuth, ok := other.(*JWTAuthenticator)
	if !ok {
		return false
	}
	return auth.Username == otherAuth.Username &&
		auth.Password == otherAuth.Password &&
		auth.Host == otherAuth.Host &&
		auth.Port == otherAuth.Port &&
		auth.Tenant == otherAuth.Tenant &&
		auth.SslVerify == otherAuth.SslVerify
}

func (auth *JWTAuthenticator) setInitialized(state bool) {
	auth.mu.Lock()
	defer auth.mu.Unlock()
	auth.initialized = state
}

func (auth *JWTAuthenticator) isInitialized() bool {
	auth.mu.RLock()
	defer auth.mu.RUnlock()
	return auth.initialized
}

type ApiRTokenAuthenticator struct {
	Host      string
	Port      uint64
	SslVerify bool
	Token     string
	Tenant    string
}

func (auth *ApiRTokenAuthenticator) authorize() error {
	// No-op for ApiRTokenAuthenticator
	return nil
}

func (auth *ApiRTokenAuthenticator) setAuthHeader(headers *http.Header) {
	headers.Add("Authorization", "Api-Token "+auth.Token)
	if auth.Tenant != "" {
		headers.Add(HeaderXTenantName, auth.Tenant)
	}
}

func (auth *ApiRTokenAuthenticator) equal(other Authenticator) bool {
	otherAuth, ok := other.(*ApiRTokenAuthenticator)
	if !ok {
		return false
	}
	return auth.Token == otherAuth.Token &&
		auth.Host == otherAuth.Host &&
		auth.Port == otherAuth.Port &&
		auth.Tenant == otherAuth.Tenant &&
		auth.SslVerify == otherAuth.SslVerify
}

func (auth *ApiRTokenAuthenticator) setInitialized(_ bool) {
	// No-op
}

func (auth *ApiRTokenAuthenticator) isInitialized() bool {
	// ApiToken is always "initialized" - no auth call needed
	return true
}

type BaseAuthAuthenticator struct {
	Host        string
	Port        uint64
	SslVerify   bool
	Username    string
	Password    string
	Tenant      string
	encodedAuth string // Cached Base64-encoded credentials
}

func (auth *BaseAuthAuthenticator) authorize() error {
	// Pre-compute and cache the Base64-encoded Basic Auth credentials
	// This is called once during setup, avoiding repeated encoding on each request
	authStr := auth.Username + ":" + auth.Password
	auth.encodedAuth = base64.StdEncoding.EncodeToString([]byte(authStr))
	return nil
}

func (auth *BaseAuthAuthenticator) setAuthHeader(headers *http.Header) {
	// Use the pre-encoded credentials from authorize()
	headers.Add(HeaderAuthorization, AuthTypeBasic+" "+auth.encodedAuth)
	if auth.Tenant != "" {
		headers.Add(HeaderXTenantName, auth.Tenant)
	}
}

func (auth *BaseAuthAuthenticator) equal(other Authenticator) bool {
	otherAuth, ok := other.(*BaseAuthAuthenticator)
	if !ok {
		return false
	}
	return auth.Username == otherAuth.Username &&
		auth.Password == otherAuth.Password &&
		auth.Host == otherAuth.Host &&
		auth.Port == otherAuth.Port &&
		auth.Tenant == otherAuth.Tenant &&
		auth.SslVerify == otherAuth.SslVerify
}

func (auth *BaseAuthAuthenticator) setInitialized(_ bool) {
	// No-op for Basic Auth
}

func (auth *BaseAuthAuthenticator) isInitialized() bool {
	// Basic Auth just encodes credentials, always ready after creation
	return auth.encodedAuth != ""
}
