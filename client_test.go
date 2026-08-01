package vast_client

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func parseTestServerAddress(addr string) (host string, port uint64) {
	lastColon := strings.LastIndex(addr, ":")
	if lastColon == -1 {
		return addr, 443
	}
	host = addr[:lastColon]
	portNum, _ := strconv.ParseUint(addr[lastColon+1:], 10, 64)
	return host, portNum
}

func testVMSConfig(t *testing.T, server *httptest.Server) *VMSConfig {
	t.Helper()
	host, port := parseTestServerAddress(server.Listener.Addr().String())
	timeout := time.Minute
	return &VMSConfig{
		Host:           host,
		Port:           port,
		ApiToken:       "test-token",
		SslVerify:      false,
		Timeout:        &timeout,
		MaxConnections: 5,
		ApiVersion:     "latest",
		UserAgent:      "go-vast-client-test/1.0",
	}
}

func TestNewVMSRest_InvalidConfig(t *testing.T) {
	if _, err := NewVMSRest(&VMSConfig{}); err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestNewTypedVMSRest_InvalidConfig(t *testing.T) {
	if _, err := NewTypedVMSRest(&VMSConfig{}); err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestNewVMSRest_CreatesClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewVMSRest(testVMSConfig(t, server))
	if err != nil {
		t.Fatalf("NewVMSRest: %v", err)
	}
	if client.Users == nil {
		t.Fatal("expected Users resource on untyped client")
	}
}

func TestNewTypedVMSRest_CreatesClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewTypedVMSRest(testVMSConfig(t, server))
	if err != nil {
		t.Fatalf("NewTypedVMSRest: %v", err)
	}
	if client.Users == nil {
		t.Fatal("expected Users resource on typed client")
	}
}

func TestErrorHelpersReexported(t *testing.T) {
	if IsNotFoundErr == nil || IgnoreNotFound == nil || IsApiError == nil {
		t.Fatal("expected error helpers to be re-exported")
	}
}
