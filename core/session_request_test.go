package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

// newTestSession creates a VMSSession backed by the given TLS httptest.Server
// using an API-token so no JWT handshake is required.
func newTestSession(t *testing.T, server *httptest.Server) *VMSSession {
	t.Helper()
	host, port := parseTestServerAddress(server.Listener.Addr().String())
	timeout := time.Minute
	cfg := &VMSConfig{
		Host:           host,
		Port:           port,
		ApiToken:       "test-token",
		SslVerify:      false,
		Timeout:        &timeout,
		MaxConnections: 5,
		ApiVersion:     "latest",
	}
	session, err := NewVMSSession(cfg)
	if err != nil {
		t.Fatalf("NewVMSSession: %v", err)
	}
	return session
}

// newTestResource returns a Dummy resource backed by the given session.
// Using the "Dummy" resource type bypasses the @resourceType annotation that
// doAfterRequest normally attaches, keeping test assertions simple.
func newTestResource(session *VMSSession) *Dummy {
	return NewDummy(context.Background(), session)
}

// TestRequest_PaginatedObjectUnwrapped verifies that when the API returns a
// Django-REST-Framework paginated JSON object
//
//	{"count":N,"next":…,"previous":…,"results":[…]}
//
// and the caller requests RecordSet, RequestWithHeaders unpacks the "results"
// array into a flat RecordSet instead of wrapping the entire wrapper object as
// a single element.
func TestRequest_PaginatedObjectUnwrapped(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		want    RecordSet
	}{
		{
			name: "two records",
			payload: map[string]any{
				"count":    2,
				"next":     nil,
				"previous": nil,
				"results": []any{
					map[string]any{"id": 1, "key": "alpha"},
					map[string]any{"id": 2, "key": "beta"},
				},
			},
			want: RecordSet{
				{"id": float64(1), "key": "alpha"},
				{"id": float64(2), "key": "beta"},
			},
		},
		{
			name: "single record",
			payload: map[string]any{
				"count":    1,
				"next":     nil,
				"previous": nil,
				"results": []any{
					map[string]any{"id": 42, "key": "only"},
				},
			},
			want: RecordSet{
				{"id": float64(42), "key": "only"},
			},
		},
		{
			name: "empty results list",
			payload: map[string]any{
				"count":    0,
				"next":     nil,
				"previous": nil,
				"results":  []any{},
			},
			want: RecordSet{},
		},
		{
			name: "nested fields preserved",
			payload: map[string]any{
				"count":    1,
				"next":     nil,
				"previous": nil,
				"results": []any{
					map[string]any{
						"id":   7,
						"meta": map[string]any{"active": true, "tags": []any{"a", "b"}},
					},
				},
			},
			want: RecordSet{
				{
					"id":   float64(7),
					"meta": map[string]any{"active": true, "tags": []any{"a", "b"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.payload)
			}))
			defer server.Close()

			session := newTestSession(t, server)
			resource := newTestResource(session)

			got, err := Request[RecordSet](context.Background(), resource, http.MethodGet, "/test/", nil, nil)
			if err != nil {
				t.Fatalf("Request: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("result mismatch:\ngot:  %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

// TestRequest_BareArrayAsRecordSet verifies that when the API returns a bare
// JSON array (not wrapped in a paginated object), Request[RecordSet] still
// returns a correct flat RecordSet.
func TestRequest_BareArrayAsRecordSet(t *testing.T) {
	payload := []map[string]any{
		{"id": 1, "name": "alice"},
		{"id": 2, "name": "bob"},
	}
	want := RecordSet{
		{"id": float64(1), "name": "alice"},
		{"id": float64(2), "name": "bob"},
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	session := newTestSession(t, server)
	resource := newTestResource(session)

	got, err := Request[RecordSet](context.Background(), resource, http.MethodGet, "/test/", nil, nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("result mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

// TestRequest_SingleObjectAsRecord verifies that when the API returns a plain
// JSON object (not a paginated wrapper), Request[Record] returns it as-is.
func TestRequest_SingleObjectAsRecord(t *testing.T) {
	payload := map[string]any{"id": 5, "key": "environment", "default_value": "prod"}
	want := Record{"id": float64(5), "key": "environment", "default_value": "prod"}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	session := newTestSession(t, server)
	resource := newTestResource(session)

	got, err := Request[Record](context.Background(), resource, http.MethodGet, "/test/", nil, nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("result mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

// TestRequest_SingleObjectRequestedAsRecordSet verifies that when the API
// returns a plain JSON object that does NOT have a "results" key (i.e. it is
// a genuine single-record response, not a paginated wrapper), and the caller
// requests RecordSet, the single record is still wrapped as RecordSet{record}.
func TestRequest_SingleObjectRequestedAsRecordSet(t *testing.T) {
	payload := map[string]any{"id": 3, "name": "alice"}
	want := RecordSet{{"id": float64(3), "name": "alice"}}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	session := newTestSession(t, server)
	resource := newTestResource(session)

	got, err := Request[RecordSet](context.Background(), resource, http.MethodGet, "/test/", nil, nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("result mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}
