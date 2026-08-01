package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiError_ErrorVariants(t *testing.T) {
	base := &ApiError{Method: "GET", URL: "http://example", StatusCode: 404, Body: "missing"}
	if base.Error() == "" {
		t.Fatal("expected error string")
	}

	withHints := &ApiError{
		Method: "GET", URL: "http://example", StatusCode: 404, Body: "missing", hints: "resource hints",
	}
	if !stringsContains(withHints.Error(), "Resource details") {
		t.Fatalf("expected hints in error: %s", withHints.Error())
	}

	zeroStatus := &ApiError{Body: "raw body"}
	if !stringsContains(zeroStatus.Error(), "response body") {
		t.Fatalf("unexpected zero-status error: %s", zeroStatus.Error())
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestIgnoreAndExpectStatusCodes(t *testing.T) {
	apiErr := &ApiError{StatusCode: http.StatusNotFound}
	if IgnoreStatusCodes(apiErr, http.StatusNotFound) != nil {
		t.Fatal("expected nil when ignoring matching status")
	}
	if IgnoreStatusCodes(apiErr, http.StatusBadRequest) != apiErr {
		t.Fatal("expected original error when status does not match")
	}
	if IgnoreStatusCodes(errors.New("plain"), http.StatusNotFound) == nil {
		t.Fatal("expected plain error to pass through")
	}

	if !ExpectStatusCodes(apiErr, http.StatusNotFound) {
		t.Fatal("expected true for matching status")
	}
	if ExpectStatusCodes(errors.New("plain"), http.StatusNotFound) {
		t.Fatal("expected false for non-api error")
	}
}

func TestVMSSession_HTTPMethods(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"method": r.Method, "path": r.URL.Path})
	}))
	defer server.Close()

	session := newTestSession(t, server)
	ctx := context.Background()
	body := Params{"name": "test"}

	for _, call := range []struct {
		name string
		fn   func() (Renderable, error)
	}{
		{"Post", func() (Renderable, error) { return session.Post(ctx, "/items/", body, nil) }},
		{"Put", func() (Renderable, error) { return session.Put(ctx, "/items/1/", body, nil) }},
		{"Patch", func() (Renderable, error) { return session.Patch(ctx, "/items/1/", body, nil) }},
		{"Delete", func() (Renderable, error) { return session.Delete(ctx, "/items/1/", body, nil) }},
	} {
		t.Run(call.name, func(t *testing.T) {
			result, err := call.fn()
			if err != nil {
				t.Fatalf("%s: %v", call.name, err)
			}
			record, ok := result.(Record)
			if !ok {
				t.Fatalf("expected Record, got %T", result)
			}
			if record["method"] == nil {
				t.Fatalf("unexpected record: %v", record)
			}
		})
	}
}

func TestMust_PanicsAndReturns(t *testing.T) {
	if got := Must(42, nil); got != 42 {
		t.Fatalf("Must = %d", got)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic from Must")
		}
	}()
	_ = Must(0, errors.New("boom"))
}

func TestBuildResourcePathWithID(t *testing.T) {
	if got := BuildResourcePathWithID("/users", 42); got != "/users/42" {
		t.Fatalf("got %q", got)
	}
	if got := BuildResourcePathWithID("/users", "uuid-1", "tenant_data"); got != "/users/uuid-1/tenant_data" {
		t.Fatalf("got %q", got)
	}
}

func TestStructToMapAndZeroValues(t *testing.T) {
	type Nested struct {
		Flag bool `json:"flag"`
	}
	type Body struct {
		Name   string `json:"name"`
		Count  int    `json:"count,omitempty"`
		Nested Nested `json:"nested,omitempty"`
	}

	m := structToMap(Body{Name: "alice", Nested: Nested{Flag: true}})
	if m["name"] != "alice" {
		t.Fatalf("unexpected map: %v", m)
	}
	if isZeroValueInterface(nil) != true {
		t.Fatal("nil should be zero")
	}
	if isZeroValueInterface("x") != false {
		t.Fatal("non-empty string should not be zero")
	}
}
