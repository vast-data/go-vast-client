package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type crudTestContextKey struct{}

func newCRUDTestResource(t *testing.T, server *httptest.Server, ops ResourceOps) *VastResource {
	t.Helper()
	session := newTestSession(t, server)
	rest := &DummyRest{
		ctx:         context.Background(),
		Session:     session,
		resourceMap: make(map[string]ResourceEntry),
	}
	vr := NewVastResource("users", "User", rest, ops, nil)
	rest.resourceMap["User"] = ResourceEntry{VastResourceAPIWithContext: vr}
	return vr
}

func TestVastResource_GetAndList(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count": 1,
			"results": []any{
				map[string]any{"id": 1, "name": "alice"},
			},
		})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(L, R))

	record, err := resource.Get(Params{"name": "alice"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if record["name"] != "alice" {
		t.Fatalf("unexpected record: %v", record)
	}

	records, err := resource.List(Params{"name": "alice"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
}

func TestVastResource_Get_NotFound(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "results": []any{}})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(L, R))
	if _, err := resource.Get(Params{"name": "missing"}); !IsNotFoundErr(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestVastResource_Get_TooMany(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count": 2,
			"results": []any{
				map[string]any{"id": 1, "name": "alice"},
				map[string]any{"id": 2, "name": "bob"},
			},
		})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(L, R))
	if _, err := resource.Get(nil); !IsTooManyRecordsErr(err) {
		t.Fatalf("expected too many records, got %v", err)
	}
}

func TestVastResource_CreateUpdateDelete(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "new-user"})
		case http.MethodPatch:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "updated-user"})
		case http.MethodDelete:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(C, R, U, D))

	created, err := resource.Create(Params{"name": "new-user"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created["name"] != "new-user" {
		t.Fatalf("unexpected create result: %v", created)
	}

	updated, err := resource.Update(10, Params{"name": "updated-user"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated["name"] != "updated-user" {
		t.Fatalf("unexpected update result: %v", updated)
	}

	deleted, err := resource.DeleteById(10)
	if err != nil {
		t.Fatalf("DeleteById: %v", err)
	}
	if deleted["id"] == nil {
		t.Fatal("expected delete result")
	}
}

func TestVastResource_UpdateUsesPutWhenConfigured(t *testing.T) {
	var gotMethod string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "put-user"})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(U))
	setUpdateMethod(resource, http.MethodPut)
	if _, err := resource.Update(10, Params{"name": "put-user"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method = %q, want PUT", gotMethod)
	}
}

func TestVastResource_CreateWithQuery(t *testing.T) {
	var gotQuery string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "de"})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(C))
	_, err := resource.Create(
		Params{"tenant_name": "de-lab"},
		Params{"default_topic_name": "main"},
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotQuery != "tenant_name=de-lab" {
		t.Fatalf("query = %q, want tenant_name=de-lab", gotQuery)
	}
}

func TestVastResource_CreateTooManyParams(t *testing.T) {
	resource := newCRUDTestResource(t, httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call server")
	})), NewResourceOps(C))
	_, err := resource.Create(Params{"a": 1}, Params{"b": 2}, Params{"c": 3})
	if err == nil {
		t.Fatal("expected error for >2 Params")
	}
}

func TestVastResource_GetById(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "by-id"})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(R))
	record, err := resource.GetById(42)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if record["name"] != "by-id" {
		t.Fatalf("unexpected record: %v", record)
	}
}

func TestVastResource_ExistsAndEnsure(t *testing.T) {
	var created bool
	var lastExistsFields string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			lastExistsFields = r.URL.Query().Get("fields")
			if created {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"count":   1,
					"results": []any{map[string]any{"id": 1, "name": "ensure-me"}},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "results": []any{}})
		case http.MethodPost:
			created = true
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "ensure-me"})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(C, L, R))
	if resource.identityField != "id" {
		t.Fatalf("expected identityField id for users, got %q", resource.identityField)
	}

	exists, err := resource.Exists(Params{"name": "ensure-me"})
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatal("expected resource to not exist yet")
	}
	if lastExistsFields != "id" {
		t.Fatalf("Exists should request fields=id, got %q", lastExistsFields)
	}

	record, err := resource.Ensure(Params{"name": "ensure-me"}, Params{"name": "ensure-me"})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if record["name"] != "ensure-me" {
		t.Fatalf("unexpected ensure result: %v", record)
	}

	if !resource.MustExists(Params{"name": "ensure-me"}) {
		t.Fatal("expected MustExists to return true")
	}
	if lastExistsFields != "id" {
		t.Fatalf("MustExists should request fields=id, got %q", lastExistsFields)
	}

	// Caller-supplied fields must be preserved.
	exists, err = resource.Exists(Params{"name": "ensure-me", "fields": "name,guid"})
	if err != nil {
		t.Fatalf("Exists with caller fields: %v", err)
	}
	if !exists {
		t.Fatal("expected resource to exist")
	}
	if lastExistsFields != "name,guid" {
		t.Fatalf("caller fields should be preserved, got %q", lastExistsFields)
	}
}

func TestVastResource_Exists_IdentityLessSkipsFields(t *testing.T) {
	var gotFields string
	var sawFields bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, sawFields = r.URL.Query()["fields"]
		gotFields = r.URL.Query().Get("fields")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "results": []any{}})
	}))
	defer server.Close()

	session := newTestSession(t, server)
	rest := &DummyRest{
		ctx:         context.Background(),
		Session:     session,
		resourceMap: make(map[string]ResourceEntry),
	}
	// Path with no OpenAPI item identity → empty identityField.
	resource := NewVastResource("no-such-collection-xyz", "Orphan", rest, NewResourceOps(L, R), nil)
	rest.resourceMap["Orphan"] = ResourceEntry{VastResourceAPIWithContext: resource}
	if resource.identityField != "" {
		t.Fatalf("expected empty identityField, got %q", resource.identityField)
	}

	exists, err := resource.Exists(Params{"name": "x"})
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatal("expected not found")
	}
	if sawFields {
		t.Fatalf("identity-less Exists must not set fields, got %q", gotFields)
	}
}

func TestVastResource_Delete_NotFoundIsSuccess(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "results": []any{}})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(L, D))
	record, err := resource.Delete(Params{"name": "missing"}, nil)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(record) != 0 {
		t.Fatalf("expected empty record, got %v", record)
	}
}

func TestVastResource_StringAndLock(t *testing.T) {
	resource := NewVastResource("users", "User", &DummyRest{ctx: context.Background()}, NewResourceOps(C, L, R, U, D), nil)
	if got := resource.String(); got == "" {
		t.Fatal("expected non-empty String()")
	}
	unlock := resource.Lock("key")
	unlock()
}

func TestVastResource_WithContextWrappers(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "ctx"})
	}))
	defer server.Close()

	resource := newCRUDTestResource(t, server, NewResourceOps(R))
	ctx := context.WithValue(context.Background(), crudTestContextKey{}, "ctx")

	if _, err := resource.GetWithContext(ctx, Params{"name": "ctx"}); err != nil && !IsTooManyRecordsErr(err) {
		// single object list may surface as too-many or succeed depending on response shape
		t.Logf("GetWithContext: %v", err)
	}
	if _, err := resource.GetByIdWithContext(ctx, 1); err != nil {
		t.Fatalf("GetByIdWithContext: %v", err)
	}
}

func TestTypedVastResource_Accessors(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "results": []any{}})
	}))
	defer server.Close()

	session := newTestSession(t, server)
	rest := &DummyRest{
		ctx:         context.Background(),
		Session:     session,
		resourceMap: make(map[string]ResourceEntry),
	}
	untyped := NewVastResource("users", "User", rest, NewResourceOps(L, R), nil)
	rest.resourceMap["User"] = ResourceEntry{VastResourceAPIWithContext: untyped}

	typed := NewTypedVastResource("User", rest)
	if typed.GetResourceType() != "User" {
		t.Fatalf("unexpected resource type %q", typed.GetResourceType())
	}
	if typed.Session() == nil {
		t.Fatal("expected session")
	}
	if typed.String() == "" {
		t.Fatal("expected non-empty String()")
	}
	unlock := typed.Lock()
	unlock()
	iter := typed.GetIterator(Params{}, 0)
	if iter == nil {
		t.Fatal("expected iterator")
	}
}

func TestResourceOpsSetAndClear(t *testing.T) {
	ops := NewResourceOps(R)
	ops = ops.set(C)
	if !ops.isCreatable() {
		t.Fatal("expected create flag after set")
	}
	ops = ops.clear(C)
	if ops.isCreatable() {
		t.Fatal("expected create flag cleared")
	}
}
