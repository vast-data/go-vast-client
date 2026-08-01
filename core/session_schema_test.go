package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVMSSession_fetchSchema(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/token/" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access":  "access-token",
				"refresh": "refresh-token",
			})
			return
		}
		if !strings.Contains(r.URL.RawQuery, "format=openapi") {
			t.Fatalf("expected format=openapi query, got %q on %s", r.URL.RawQuery, r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != ContentTypeOpenAPI {
			t.Fatalf("Accept = %q, want %q", got, ContentTypeOpenAPI)
		}
		if r.Header.Get("Authorization") == "" {
			t.Fatal("expected Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"openapi": "3.0.0", "info": map[string]any{"title": "test"}})
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	timeout := time.Minute
	cfg := &VMSConfig{
		Host:       host,
		Port:       port,
		Username:   "admin",
		Password:   "secret",
		SslVerify:  false,
		Timeout:    &timeout,
		ApiVersion: "latest",
	}
	session, err := NewVMSSession(cfg)
	if err != nil {
		t.Fatalf("NewVMSSession: %v", err)
	}

	result, err := session.fetchSchema(context.Background())
	if err != nil {
		t.Fatalf("fetchSchema: %v", err)
	}
	if result == nil {
		t.Fatal("expected schema response")
	}
}
