package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestVastResource_DeleteWithContext_Found(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"count": 1, "results": []any{map[string]any{"id": 5, "name": "delete-me"}},
			})
		case http.MethodDelete:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(L, D))
	record, err := resource.Delete(Params{"name": "delete-me"}, Params{"force": true})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if record["id"] == nil {
		t.Fatalf("unexpected delete result: %v", record)
	}
}

func TestVastResource_EnsureWithContext_Existing(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count": 1, "results": []any{map[string]any{"id": 1, "name": "exists"}},
		})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(C, L, R))
	record, err := resource.Ensure(Params{"name": "exists"}, Params{"name": "exists"})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if record["name"] != "exists" {
		t.Fatalf("unexpected record: %v", record)
	}
}

func TestDummyRest_SetCtx(t *testing.T) {
	rest := &DummyRest{ctx: context.Background()}
	ctx := context.WithValue(context.Background(), struct{}{}, "ctx")
	rest.SetCtx(ctx)
	if rest.GetCtx() != ctx {
		t.Fatal("SetCtx did not update DummyRest context")
	}
}

func TestTypedVastResource_GetIteratorWithContext(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "results": []any{}})
	}))
	defer server.Close()

	session := newTestSession(t, server)
	rest := &DummyRest{
		ctx:         context.Background(),
		Session:     session,
		resourceMap: make(map[string]VastResourceAPIWithContext),
	}
	untyped := NewVastResource("users", "User", rest, NewResourceOps(L), nil)
	rest.resourceMap["User"] = untyped
	typed := NewTypedVastResource("User", rest)

	ctx := context.WithValue(context.Background(), struct{}{}, "iter")
	iter := typed.GetIteratorWithContext(ctx, Params{}, 10)
	if iter == nil {
		t.Fatal("expected iterator")
	}
}

func TestIsZeroValue_AllKinds(t *testing.T) {
	cases := []struct {
		name  string
		value any
		zero  bool
	}{
		{"bool false", false, true},
		{"bool true", true, false},
		{"int zero", 0, true},
		{"int nonzero", 1, false},
		{"string empty", "", true},
		{"string value", "x", false},
		{"ptr nil", (*int)(nil), true},
		{"slice nil", []int(nil), true},
		{"struct zero", struct{}{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := reflect.ValueOf(tc.value)
			if got := isZeroValue(v); got != tc.zero {
				t.Fatalf("isZeroValue(%v) = %v, want %v", tc.value, got, tc.zero)
			}
		})
	}
}

func TestStructToMap_NestedAndOmitempty(t *testing.T) {
	type Inner struct {
		Flag bool `json:"flag"`
	}
	type Outer struct {
		Name  string `json:"name"`
		Inner Inner  `json:"inner"`
		Empty string `json:"empty,omitempty"`
	}
	m := structToMap(Outer{Name: "x", Inner: Inner{Flag: true}})
	if inner, ok := m["inner"].(map[string]interface{}); !ok || inner["flag"] != true {
		t.Fatalf("unexpected nested map: %v", m["inner"])
	}
}

func TestIterator_UninitializedHasNext(t *testing.T) {
	mockSession := &mockSessionForIterator{responses: map[string]Renderable{}}
	mockRest := &DummyRest{ctx: context.Background(), Session: mockSession}
	mockResource := &mockResourceForIterator{
		VastResource: &VastResource{resourcePath: "resources", resourceType: "TestResource", Rest: mockRest},
		mockSession:  mockSession,
	}
	iter := NewResourceIterator(context.Background(), mockResource, Params{}, 10)
	if !iter.HasNext() {
		t.Fatal("uninitialized iterator should report HasNext=true")
	}
	if iter.HasPrevious() {
		t.Fatal("uninitialized iterator should not have previous page")
	}
	if _, err := iter.Previous(); err == nil {
		t.Fatal("expected error when calling Previous before Next")
	}
}

func TestStructToMap_PointerAndNil(t *testing.T) {
	type Body struct {
		Name  string  `json:"name"`
		Count int     `json:"count,omitempty"`
		Flag  *bool   `json:"flag,omitempty"`
		Skip  *string `json:"-"`
	}
	flag := true
	m := structToMap(&Body{Name: "alice", Flag: &flag})
	if m["name"] != "alice" {
		t.Fatalf("unexpected map: %v", m)
	}
	if _, ok := m["skip"]; ok {
		t.Fatal("expected dash tag to be skipped")
	}
	if len(structToMap(nil)) != 0 {
		t.Fatal("nil should produce empty map")
	}
}

func TestFormatPathParamValue(t *testing.T) {
	if got := formatPathParamValue(int64(7)); got != "7" {
		t.Fatalf("got %q", got)
	}
	if got := formatPathParamValue("uuid"); got != "uuid" {
		t.Fatalf("got %q", got)
	}
}

func TestDoRequestWithRetries_PermissionDeniedStopsRetry(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"detail":"permission_denied"}`))
	}))
	defer server.Close()

	session := newTestSession(t, server)
	_, err := session.Get(context.Background(), "/items/", nil, nil)
	if err == nil {
		t.Fatal("expected permission error")
	}
}

func TestDoAfterRequest_AfterRequestFnError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(R))
	resource.Session().GetConfig().AfterRequestFn = func(ctx context.Context, response Renderable) (Renderable, error) {
		return nil, errors.New("after hook failed")
	}
	_, err := resource.GetById(1)
	if err == nil {
		t.Fatal("expected after-request error")
	}
}

func TestDefaultResponseMutations_RecordSet(t *testing.T) {
	got, err := defaultResponseMutations(RecordSet{{"id": 1}})
	if err != nil {
		t.Fatalf("defaultResponseMutations: %v", err)
	}
	if len(got.(RecordSet)) != 1 {
		t.Fatal("expected record set passthrough")
	}
}

func TestParseToken_InvalidJSON(t *testing.T) {
	rsp := createMockResponse("not-json", 200, 8)
	if _, err := parseToken(rsp); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestIsZeroValue_MoreKinds(t *testing.T) {
	cases := []struct {
		name  string
		value any
		zero  bool
	}{
		{"float32 zero", float32(0), true},
		{"float64 zero", float64(0), true},
		{"complex zero", complex(0, 0), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isZeroValue(reflect.ValueOf(tc.value)); got != tc.zero {
				t.Fatalf("isZeroValue = %v", got)
			}
		})
	}
}

func TestApplyCallbackForRecordUnion_RecordSet(t *testing.T) {
	set := RecordSet{{"id": 1}}
	got, err := applyCallbackForRecordUnion[RecordSet](set, func(r Renderable) (Renderable, error) {
		return r, nil
	})
	if err != nil {
		t.Fatalf("applyCallbackForRecordUnion: %v", err)
	}
	if len(got.(RecordSet)) != 1 {
		t.Fatal("expected record set callback")
	}
}

func TestApplyCallbackForRecordUnion_TypeMismatch(t *testing.T) {
	rec := Record{"id": 1}
	got, err := applyCallbackForRecordUnion[RecordSet](rec, func(r Renderable) (Renderable, error) {
		t.Fatal("callback should not run for type mismatch")
		return r, nil
	})
	if err != nil {
		t.Fatalf("applyCallbackForRecordUnion: %v", err)
	}
	if len(got.(Record)) != 1 {
		t.Fatal("expected passthrough record")
	}

	set := RecordSet{{"id": 1}}
	got, err = applyCallbackForRecordUnion[Record](set, func(r Renderable) (Renderable, error) {
		t.Fatal("callback should not run for type mismatch")
		return r, nil
	})
	if err != nil {
		t.Fatalf("applyCallbackForRecordUnion: %v", err)
	}
	if len(got.(RecordSet)) != 1 {
		t.Fatal("expected passthrough record set")
	}
}

func TestApplyCallbackForRecordUnion_UnsupportedType(t *testing.T) {
	_, err := applyCallbackForRecordUnion[Record](nil, func(r Renderable) (Renderable, error) {
		return r, nil
	})
	if err == nil {
		t.Fatal("expected unsupported type error")
	}
}

func TestDoRequest_Multipart(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	session := newTestSession(t, server)
	headers := []http.Header{{HeaderContentType: {ContentTypeMultipartForm}}}
	body := Params{
		"name": "alice",
		"file": FileData{Filename: "a.txt", Content: []byte("data")},
	}
	if _, err := session.Post(context.Background(), "/upload/", body, headers); err != nil {
		t.Fatalf("Post multipart: %v", err)
	}
}
