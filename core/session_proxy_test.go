package core

import (
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

// clearProxyEnv clears all proxy environment variables and registers cleanup to restore them
func clearProxyEnv(t *testing.T) {
	t.Helper()
	proxyVars := []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy"}

	// Save original values
	saved := make(map[string]string)
	for _, key := range proxyVars {
		if val := os.Getenv(key); val != "" {
			saved[key] = val
		}
		os.Unsetenv(key)
	}

	// Register cleanup to restore original values
	t.Cleanup(func() {
		// First clear everything again (in case test set something)
		for _, key := range proxyVars {
			os.Unsetenv(key)
		}
		// Then restore originals
		for key, val := range saved {
			os.Setenv(key, val)
		}
	})
}

// TestProxyEnvironmentVariable verifies that HTTPS_PROXY environment variable is respected
func TestProxyEnvironmentVariable(t *testing.T) {
	// Set up test proxy URL
	testProxyURL := "http://10.100.11.154:3128"

	// Clear all proxy environment variables first (handles cleanup automatically)
	clearProxyEnv(t)

	// Set the proxy environment variable we want to test
	os.Setenv("HTTPS_PROXY", testProxyURL)

	// Create a test config using API token to avoid authentication
	timeout := time.Minute
	config := &VMSConfig{
		Host:           "test.example.com",
		Port:           443,
		ApiToken:       "test-token",
		SslVerify:      false,
		Timeout:        &timeout,
		MaxConnections: 10,
	}

	// Create a session
	session, err := NewVMSSession(config)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify that the transport has the proxy set
	transport, ok := session.client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected http.Transport, got different type")
	}

	if transport.Proxy == nil {
		t.Fatal("Proxy function is nil - environment variables will not be respected")
	}

	// Test that the proxy function returns the expected proxy URL
	testURL, _ := url.Parse("https://test.example.com")
	req := &http.Request{URL: testURL}
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("Proxy function returned error: %v", err)
	}

	if proxyURL == nil {
		t.Fatal("Proxy function returned nil - HTTPS_PROXY not being respected")
	}

	if proxyURL.String() != testProxyURL {
		t.Errorf("Expected proxy URL %s, got %s", testProxyURL, proxyURL.String())
	}
}

// TestNO_PROXY verifies that NO_PROXY environment variable is respected
// Note: Due to Go's internal proxy caching in http.ProxyFromEnvironment, this test only
// passes when run in isolation. To run it, use:
//
//	go test ./core -run ^TestNO_PROXY$ -v
func TestNO_PROXY(t *testing.T) {
	// Skip by default to avoid test suite failures due to Go's proxy caching
	// Only run when explicitly requested
	if os.Getenv("RUN_ISOLATION_TESTS") == "" {
		t.Skip("Skipping TestNO_PROXY (requires isolation). To run: go test ./core -run ^TestNO_PROXY$ -v")
	}

	// Set up test proxy and NO_PROXY
	testProxyURL := "http://proxy.example.com:8080"

	// Clear all proxy environment variables first (handles cleanup automatically)
	clearProxyEnv(t)

	// Set the proxy and NO_PROXY environment variables we want to test
	os.Setenv("HTTPS_PROXY", testProxyURL)
	os.Setenv("NO_PROXY", "localhost,127.0.0.1,internal.example.com")

	// Create a test config using API token to avoid authentication
	timeout := time.Minute
	config := &VMSConfig{
		Host:           "test.example.com",
		Port:           443,
		ApiToken:       "test-token",
		SslVerify:      false,
		Timeout:        &timeout,
		MaxConnections: 10,
	}

	// Create a session
	session, err := NewVMSSession(config)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	transport, ok := session.client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected http.Transport, got different type")
	}

	// Test that proxy is used for non-excluded hosts
	testURL1, _ := url.Parse("https://external.example.com")
	req1 := &http.Request{URL: testURL1}
	proxyURL1, _ := transport.Proxy(req1)
	if proxyURL1 == nil || proxyURL1.String() != testProxyURL {
		t.Errorf("Expected proxy for external host, got %v", proxyURL1)
	}

	// Test that proxy is NOT used for excluded hosts
	testURL2, _ := url.Parse("https://internal.example.com")
	req2 := &http.Request{URL: testURL2}
	proxyURL2, _ := transport.Proxy(req2)
	if proxyURL2 != nil {
		t.Errorf("Expected no proxy for NO_PROXY host, got %v", proxyURL2)
	}
}
