package core

import (
	"context"
	"fmt"
	"testing"
)

func conventionalURL(resource string, id int) string {
	return fmt.Sprintf("https://test.example.com:443/api/v5/%s/%d/", resource, id)
}

func assertRecordDisplayName(t *testing.T, record Record, want string) {
	t.Helper()
	if got := recordDisplayName(record); got != want {
		t.Fatalf("recordDisplayName = %q, want %q (url=%v)", got, want, record["url"])
	}
}

// Test that iterator preserves conventional url fields on records (paginated response).
func TestIterator_SetsResourceType(t *testing.T) {
	response := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "item1", "url": conventionalURL("views", 1)},
			map[string]any{"id": float64(2), "name": "item2", "url": conventionalURL("views", 2)},
		},
		"count":    float64(2),
		"next":     nil,
		"previous": nil,
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/views/?page_size=10": response,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	viewResource := NewVastResource("/views", "View", mockRest, NewResourceOps(L), nil)
	iter := NewResourceIterator(context.Background(), viewResource, Params{}, 10)

	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}

	for i, record := range records {
		assertRecordDisplayName(t, record, "View")
		if _, ok := record[ResourceTypeKey]; ok {
			t.Errorf("Record %d should not have injected %s", i, ResourceTypeKey)
		}
	}
}

func TestIterator_SetsResourceType_TypedResults(t *testing.T) {
	response := Record{
		"results": []map[string]any{
			{"id": float64(1), "name": "snapshot1", "url": conventionalURL("snapshots", 1)},
			{"id": float64(2), "name": "snapshot2", "url": conventionalURL("snapshots", 2)},
		},
		"count":    float64(2),
		"next":     nil,
		"previous": nil,
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/snapshots/?page_size=5": response,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	snapshotResource := NewVastResource("/snapshots", "Snapshot", mockRest, NewResourceOps(L), nil)
	iter := NewResourceIterator(context.Background(), snapshotResource, Params{}, 5)

	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}

	for _, record := range records {
		assertRecordDisplayName(t, record, "Snapshot")
	}
}

func TestIterator_SetsResourceType_NonPaginated(t *testing.T) {
	response := RecordSet{
		{"id": float64(1), "name": "tenant1", "url": conventionalURL("tenants", 1)},
		{"id": float64(2), "name": "tenant2", "url": conventionalURL("tenants", 2)},
		{"id": float64(3), "name": "tenant3", "url": conventionalURL("tenants", 3)},
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/tenants/?page_size=10": response,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	tenantResource := NewVastResource("/tenants", "Tenant", mockRest, NewResourceOps(L), nil)
	iter := NewResourceIterator(context.Background(), tenantResource, Params{}, 10)

	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(records))
	}

	for _, record := range records {
		assertRecordDisplayName(t, record, "Tenant")
	}
}

func TestIterator_SetsResourceType_SingleRecord(t *testing.T) {
	response := Record{
		"id":     float64(42),
		"name":   "single-item",
		"status": "active",
		"url":    conventionalURL("items", 42),
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/items/?page_size=10": response,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	itemResource := NewVastResource("/items", "Item", mockRest, NewResourceOps(L), nil)
	iter := NewResourceIterator(context.Background(), itemResource, Params{}, 10)

	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	assertRecordDisplayName(t, records[0], "Item")
}

func TestIterator_SetsResourceType_MultiplePages(t *testing.T) {
	page1 := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "user1", "url": conventionalURL("users", 1)},
			map[string]any{"id": float64(2), "name": "user2", "url": conventionalURL("users", 2)},
		},
		"count":    float64(4),
		"next":     "https://test.example.com:443/api/v1/users/?page=2",
		"previous": nil,
	}

	page2 := Record{
		"results": []any{
			map[string]any{"id": float64(3), "name": "user3", "url": conventionalURL("users", 3)},
			map[string]any{"id": float64(4), "name": "user4", "url": conventionalURL("users", 4)},
		},
		"count":    float64(4),
		"next":     nil,
		"previous": "https://test.example.com:443/api/v1/users/?page=1",
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/users/?page_size=2": page1,
			"https://test.example.com:443/api/v1/users/?page=2":      page2,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	userResource := NewVastResource("/users", "User", mockRest, NewResourceOps(L), nil)
	iter := NewResourceIterator(context.Background(), userResource, Params{}, 2)

	records1, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error on page 1, got: %v", err)
	}

	if len(records1) != 2 {
		t.Fatalf("Expected 2 records on page 1, got %d", len(records1))
	}

	for _, record := range records1 {
		assertRecordDisplayName(t, record, "User")
	}

	records2, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error on page 2, got: %v", err)
	}

	if len(records2) != 2 {
		t.Fatalf("Expected 2 records on page 2, got %d", len(records2))
	}

	for _, record := range records2 {
		assertRecordDisplayName(t, record, "User")
	}
}

func TestIterator_SetsResourceType_All(t *testing.T) {
	page1 := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "quota1", "url": conventionalURL("quotas", 1)},
		},
		"count":    float64(2),
		"next":     "https://test.example.com:443/api/v1/quotas/?page=2",
		"previous": nil,
	}

	page2 := Record{
		"results": []any{
			map[string]any{"id": float64(2), "name": "quota2", "url": conventionalURL("quotas", 2)},
		},
		"count":    float64(2),
		"next":     nil,
		"previous": "https://test.example.com:443/api/v1/quotas/?page=1",
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/quotas/?page_size=1": page1,
			"https://test.example.com:443/api/v1/quotas/?page=2":      page2,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	quotaResource := NewVastResource("/quotas", "Quota", mockRest, NewResourceOps(L), nil)
	iter := NewResourceIterator(context.Background(), quotaResource, Params{}, 1)

	allRecords, err := iter.All()
	if err != nil {
		t.Fatalf("Expected no error from All(), got: %v", err)
	}

	if len(allRecords) != 2 {
		t.Fatalf("Expected 2 total records, got %d", len(allRecords))
	}

	for _, record := range allRecords {
		assertRecordDisplayName(t, record, "Quota")
	}
}

func TestIterator_SetsResourceType_Reset(t *testing.T) {
	response := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "policy1", "url": conventionalURL("policies", 1)},
		},
		"count":    float64(1),
		"next":     nil,
		"previous": nil,
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/policies/?page_size=10": response,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	policyResource := NewVastResource("/policies", "Policy", mockRest, NewResourceOps(L), nil)
	iter := NewResourceIterator(context.Background(), policyResource, Params{}, 10)

	if _, err := iter.Next(); err != nil {
		t.Fatalf("Expected no error on first Next(), got: %v", err)
	}

	records, err := iter.Reset()
	if err != nil {
		t.Fatalf("Expected no error from Reset(), got: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record after Reset(), got %d", len(records))
	}

	assertRecordDisplayName(t, records[0], "Policy")
}

func TestIterator_DummyResourceNoType(t *testing.T) {
	response := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "item1"},
		},
		"count":    float64(1),
		"next":     nil,
		"previous": nil,
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/dummy/?page_size=10": response,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	dummyResource := NewVastResource("/dummy", "Dummy", mockRest, 0, nil)
	iter := NewResourceIterator(context.Background(), dummyResource, Params{}, 10)

	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	for i, record := range records {
		if _, ok := record[ResourceTypeKey]; ok {
			t.Errorf("Record %d should not have injected %s", i, ResourceTypeKey)
		}
		if got := recordDisplayName(record); got != "" {
			t.Errorf("Record %d should not have display name without conventional url, got %q", i, got)
		}
	}
}
