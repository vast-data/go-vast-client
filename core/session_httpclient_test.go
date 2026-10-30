package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recordingTransport struct {
	base  http.RoundTripper
	paths []string
}

func (rt *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.paths = append(rt.paths, r.URL.Path)
	return rt.base.RoundTrip(r)
}

// TestVMSSession_HTTPClientTransportWrap verifies that replacing the Transport on
// the accessor's client puts the wrapper under every API request.
func TestVMSSession_HTTPClientTransportWrap(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer server.Close()

	session := newTestSession(t, server)
	client := session.HTTPClient()
	if client == nil {
		t.Fatal("HTTPClient returned nil")
	}
	rec := &recordingTransport{base: client.Transport}
	client.Transport = rec

	resource := newTestResource(session)
	got, err := Request[Record](context.Background(), resource, http.MethodGet, "/test/", nil, nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if got["id"] != float64(1) {
		t.Errorf("response not passed through: %+v", got)
	}
	if len(rec.paths) != 1 || rec.paths[0] != "/api/latest/test/" {
		t.Errorf("wrapper saw %v, want [/api/latest/test/]", rec.paths)
	}
}
