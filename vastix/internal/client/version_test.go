package client

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	vastclient "github.com/vast-data/go-vast-client"
)

func TestFetchVastVersion_ForbiddenReturnsPlaceholder(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case (r.URL.Path == "/api/token/" || r.URL.Path == "/api/latest/token/") && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access":  "test-access",
				"refresh": "test-refresh",
			})
		case r.Method == http.MethodGet && strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/versions"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"detail":"permission_denied","code":"permission_denied"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	host, port := splitHostPort(t, server.Listener.Addr().String())
	rest, err := vastclient.NewVMSRest(&vastclient.VMSConfig{
		Host:       host,
		Port:       port,
		Username:   "tenant-admin",
		Password:   "secret",
		SslVerify:  false,
		ApiVersion: "latest",
	})
	if err != nil {
		t.Fatalf("NewVMSRest: %v", err)
	}

	got, err := FetchVastVersion(context.Background(), rest)
	if err != nil {
		t.Fatalf("FetchVastVersion: %v", err)
	}
	if got != VastVersionUnavailable {
		t.Fatalf("got %q want %q", got, VastVersionUnavailable)
	}
}

func TestFetchVastVersion_NilRest(t *testing.T) {
	if _, err := FetchVastVersion(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil rest")
	}
}

func splitHostPort(t *testing.T, addr string) (string, uint64) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SplitHostPort(%q): %v", addr, err)
	}
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		t.Fatalf("ParseUint(%q): %v", portStr, err)
	}
	return host, port
}
