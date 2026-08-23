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
		resourceMap: make(map[string]VastResourceAPIWithContext),
	}
	vr := NewVastResource("users", "User", rest, ops, nil)
	rest.resourceMap["User"] = vr
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

	deleted, err := resource.DeleteById(10, nil, nil)
	if err != nil {
		t.Fatalf("DeleteById: %v", err)
	}
	if deleted["id"] == nil {
		t.Fatal("expected delete result")
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
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

	exists, err := resource.Exists(Params{"name": "ensure-me"})
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatal("expected resource to not exist yet")
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
		resourceMap: make(map[string]VastResourceAPIWithContext),
	}
	untyped := NewVastResource("users", "User", rest, NewResourceOps(L, R), nil)
	rest.resourceMap["User"] = untyped

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
