package core

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestApiRTokenAuthenticator_SetAuthHeaderAndEqual(t *testing.T) {
	auth := &ApiRTokenAuthenticator{
		Host:      "vms.example.com",
		Port:      443,
		SslVerify: true,
		Token:     "secret-token",
		Tenant:    "tenant-a",
	}
	other := &ApiRTokenAuthenticator{
		Host:      "vms.example.com",
		Port:      443,
		SslVerify: true,
		Token:     "secret-token",
		Tenant:    "tenant-a",
	}
	diff := &ApiRTokenAuthenticator{Token: "other"}

	headers := &http.Header{}
	auth.setAuthHeader(headers)
	if got := headers.Get("Authorization"); got != "Api-Token secret-token" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := headers.Get(HeaderXTenantName); got != "tenant-a" {
		t.Fatalf("tenant header = %q", got)
	}

	auth.setInitialized(true)
	if !auth.isInitialized() {
		t.Fatal("api token should stay initialized")
	}
	if !auth.equal(other) {
		t.Fatal("expected equal authenticators")
	}
	if auth.equal(diff) {
		t.Fatal("expected different authenticators")
	}
	if auth.equal(&JWTAuthenticator{}) {
		t.Fatal("expected false for different type")
	}
}

func TestBaseAuthAuthenticator_SetAuthHeaderAndEqual(t *testing.T) {
	auth := &BaseAuthAuthenticator{
		Host:      "vms.example.com",
		Port:      443,
		SslVerify: false,
		Username:  "admin",
		Password:  "password",
		Tenant:    "tenant-b",
	}
	if err := auth.authorize(); err != nil {
		t.Fatalf("authorize: %v", err)
	}

	headers := &http.Header{}
	auth.setAuthHeader(headers)
	if headers.Get("Authorization") == "" {
		t.Fatal("expected basic auth header")
	}
	if headers.Get(HeaderXTenantName) != "tenant-b" {
		t.Fatalf("tenant header = %q", headers.Get(HeaderXTenantName))
	}

	other := &BaseAuthAuthenticator{
		Host: "vms.example.com", Port: 443, SslVerify: false,
		Username: "admin", Password: "password", Tenant: "tenant-b",
		encodedAuth: auth.encodedAuth,
	}
	auth.setInitialized(false)
	if !auth.isInitialized() {
		t.Fatal("basic auth should remain initialized via encoded credentials")
	}
	if !auth.equal(other) {
		t.Fatal("expected equal basic auth authenticators")
	}
	if auth.equal(&ApiRTokenAuthenticator{}) {
		t.Fatal("expected false for different type")
	}
}

func TestJWTAuthenticator_SetInitialized(t *testing.T) {
	auth := &JWTAuthenticator{
		Host:     "vms.example.com",
		Port:     443,
		Username: "admin",
		Password: "password",
		Token:    &jwtToken{Access: "token"},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	auth.setInitialized(true)
	if !auth.isInitialized() {
		t.Fatal("expected initialized after setInitialized(true)")
	}
	auth.setInitialized(false)
	if auth.isInitialized() {
		t.Fatal("expected not initialized after setInitialized(false)")
	}
}

func TestJWTAuthenticator_RefreshTokenErrors(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/refresh/" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"detail":"invalid refresh"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access":  "access",
			"refresh": "refresh",
		})
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := &JWTAuthenticator{
		Host: host, Port: port, SslVerify: false,
		Username: "admin", Password: "password",
		Tenant: "tenant-a",
		Token:  &jwtToken{Access: "old", Refresh: "refresh"},
	}
	auth.authCond = sync.NewCond(&auth.mu)
	auth.setInitialized(true)

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- httptest.NewTLSServer uses a self-signed cert
	}}
	if err := auth.refreshToken(client); err == nil {
		t.Fatal("expected refresh error")
	}
}

func TestJWTAuthenticator_AcquireTokenWithTenantHeader(t *testing.T) {
	var gotPath, tenantHeader string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		tenantHeader = r.Header.Get(HeaderXTenantName)
		if r.URL.Path == "/api/token/" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access":  "access",
				"refresh": "refresh",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := &JWTAuthenticator{
		Host: host, Port: port, SslVerify: false,
		Username: "admin", Password: "password",
		Tenant: "tenant-b",
		Token:  &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- httptest.NewTLSServer uses a self-signed cert
	}}
	if err := auth.acquireToken(client); err != nil {
		t.Fatalf("acquireToken: %v", err)
	}
	if gotPath != "/api/token/" {
		t.Fatalf("token path = %q, want /api/token/", gotPath)
	}
	if tenantHeader != "tenant-b" {
		t.Fatalf("X-Tenant-Name = %q, want tenant-b", tenantHeader)
	}
}

func TestJWTAuthenticator_AcquireTokenWithTenantPath(t *testing.T) {
	var gotPath, tenantHeader string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		tenantHeader = r.Header.Get(HeaderXTenantName)
		if r.URL.Path == "/api/token/tenant-b/" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access":  "access",
				"refresh": "refresh",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	auth := &JWTAuthenticator{
		Host: host, Port: port, SslVerify: false,
		Username: "tenant-admin", Password: "password",
		Tenant:             "tenant-b",
		useTenantTokenPath: true,
		Token:              &jwtToken{},
	}
	auth.authCond = sync.NewCond(&auth.mu)

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- httptest.NewTLSServer uses a self-signed cert
	}}
	if err := auth.acquireToken(client); err != nil {
		t.Fatalf("acquireToken: %v", err)
	}
	if gotPath != "/api/token/tenant-b/" {
		t.Fatalf("token path = %q, want /api/token/tenant-b/", gotPath)
	}
	if tenantHeader != "" {
		t.Fatalf("X-Tenant-Name on obtain = %q, want empty", tenantHeader)
	}
	if auth.Token.Access != "access" {
		t.Fatalf("access token = %q", auth.Token.Access)
	}
}

func TestCreateAuthenticator_UseTenantTokenPath(t *testing.T) {
	authenticatorsMu.Lock()
	original := authenticators
	authenticators = nil
	authenticatorsMu.Unlock()
	defer func() {
		authenticatorsMu.Lock()
		authenticators = original
		authenticatorsMu.Unlock()
	}()

	auth, err := createAuthenticator(&VMSConfig{
		Host: "test.example.com", Port: 443, SslVerify: false,
		Username: "tenant-admin", Password: "password", Tenant: "tenant-b",
	})
	if err != nil {
		t.Fatalf("createAuthenticator: %v", err)
	}
	jwtAuth, ok := auth.(*JWTAuthenticator)
	if !ok {
		t.Fatalf("got %T, want *JWTAuthenticator", auth)
	}
	if jwtAuth.useTenantTokenPath {
		t.Fatal("useTenantTokenPath should be false by default")
	}

	auth2, err := createAuthenticator(&VMSConfig{
		Host: "test.example.com", Port: 443, SslVerify: false,
		Username: "tenant-admin", Password: "password", Tenant: "tenant-b",
		UseTenantTokenPath: true,
	})
	if err != nil {
		t.Fatalf("createAuthenticator: %v", err)
	}
	jwtAuth2, ok := auth2.(*JWTAuthenticator)
	if !ok {
		t.Fatalf("got %T, want *JWTAuthenticator", auth2)
	}
	if !jwtAuth2.useTenantTokenPath {
		t.Fatal("useTenantTokenPath should be true")
	}
	if auth == auth2 {
		t.Fatal("header vs path login must not share the same authenticator instance")
	}
}
