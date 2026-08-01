package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInterceptorLogging(t *testing.T) {
	original := logLevel
	t.Cleanup(func() { logLevel = original })

	logLevel = "info"
	beforeRequestLog(http.MethodGet, "http://example/api", nil)
	afterRequestLog(Record{"url": "https://l101:443/api/v5/users/1/", "id": 1})
	afterRequestLogInfo(RecordSet{{"url": "https://l101:443/api/v5/users/1/", "id": 1}})
	afterRequestLogInfo(RecordSet{})
	afterRequestLogInfo(Record{"id": 1})
	afterRequestLogInfo(nil)

	logLevel = "debug"
	body := io.NopCloser(bytes.NewBufferString(`{"name":"alice"}`))
	beforeRequestLog(http.MethodPost, "http://example/api", body)
	afterRequestLogDebug(Record{"url": "https://l101:443/api/v5/users/1/", "name": "alice"})
	afterRequestLogDebug(Record{"name": "alice"})
	afterRequestLogDebug(RecordSet{{"url": "https://l101:443/api/v5/users/1/", "id": 1}})
	afterRequestLogDebug(RecordSet{{"id": 1}})
	afterRequestLogDebug(RecordSet{})
	afterRequestLogDebug(nil)

	beforeRequestLog(http.MethodPost, "http://example/api", io.NopCloser(bytes.NewBufferString("null")))
	beforeRequestLog(http.MethodPost, "http://example/api", io.NopCloser(bytes.NewBufferString("not-json")))
}

func TestVastResource_DoBeforeAndAfterRequest(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(R))
	_, err := resource.GetById(1)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}

	original := logLevel
	logLevel = "debug"
	t.Cleanup(func() { logLevel = original })
	_, _ = resource.GetById(1)
}

func TestDoRequestWithRetries_ReauthorizesOnUnauthorized(t *testing.T) {
	var authCalls int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/token/") {
			authCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access":  "token",
				"refresh": "refresh",
			})
			return
		}
		if authCalls == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"detail":"unauthorized"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer server.Close()

	host, port := parseTestServerAddress(server.Listener.Addr().String())
	timeout := time.Minute
	cfg := &VMSConfig{
		Host:           host,
		Port:           port,
		Username:       "admin",
		Password:       "password",
		SslVerify:      false,
		Timeout:        &timeout,
		MaxConnections: 5,
		ApiVersion:     "latest",
		UserAgent:      "go-vast-client-test/1.0",
	}
	session, err := NewVMSSession(cfg)
	if err != nil {
		t.Fatalf("NewVMSSession: %v", err)
	}

	_, err = session.Get(context.Background(), "/items/", nil, nil)
	if err != nil {
		t.Fatalf("Get after reauth: %v", err)
	}
	if authCalls == 0 {
		t.Fatal("expected auth call on unauthorized response")
	}
}

func TestDoAfterRequest_AsyncTaskEnvelopePreserved(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"async_task": map[string]any{"id": 42}})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(R))
	got, err := resource.GetById(1)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if _, ok := got["async_task"]; !ok {
		t.Fatalf("expected async_task envelope, got %v", got)
	}
}
