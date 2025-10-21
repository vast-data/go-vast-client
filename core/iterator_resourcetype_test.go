package core

import (
	"context"
	"testing"
)

// Test that iterator sets resource type on records (paginated response)
func TestIterator_SetsResourceType(t *testing.T) {
	// Create mock responses
	response := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "item1"},
			map[string]any{"id": float64(2), "name": "item2"},
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

	// Create a real VastResource (not Dummy) to test resource type setting
	viewResource := NewVastResource("/views", "View", mockRest, NewResourceOps(L), nil)

	// Create iterator
	iter := NewResourceIterator(context.Background(), viewResource, Params{}, 10)

	// Get first page
	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}

	// Verify that @resourceType is set on all records
	for i, record := range records {
		resourceType, ok := record[ResourceTypeKey]
		if !ok {
			t.Errorf("Record %d missing @resourceType key", i)
		} else if resourceType != "View" {
			t.Errorf("Record %d has wrong resource type: expected 'View', got '%v'", i, resourceType)
		}
	}
}

// Test resource type with []map[string]any path (typed results)
func TestIterator_SetsResourceType_TypedResults(t *testing.T) {
	// Create mock response with []map[string]any (not []any)
	response := Record{
		"results": []map[string]any{
			{"id": float64(1), "name": "snapshot1"},
			{"id": float64(2), "name": "snapshot2"},
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

	// Verify resource type is set
	for i, record := range records {
		resourceType, ok := record[ResourceTypeKey]
		if !ok {
			t.Errorf("Record %d missing @resourceType key", i)
		} else if resourceType != "Snapshot" {
			t.Errorf("Record %d has wrong resource type: expected 'Snapshot', got '%v'", i, resourceType)
		}
	}
}

// Test resource type with non-paginated flat array response
func TestIterator_SetsResourceType_NonPaginated(t *testing.T) {
	// Create mock non-paginated response (RecordSet directly)
	response := RecordSet{
		{"id": float64(1), "name": "tenant1"},
		{"id": float64(2), "name": "tenant2"},
		{"id": float64(3), "name": "tenant3"},
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

	// Verify resource type is set on non-paginated response
	for i, record := range records {
		resourceType, ok := record[ResourceTypeKey]
		if !ok {
			t.Errorf("Record %d missing @resourceType key", i)
		} else if resourceType != "Tenant" {
			t.Errorf("Record %d has wrong resource type: expected 'Tenant', got '%v'", i, resourceType)
		}
	}
}

// Test resource type with single record response (non-pagination envelope)
func TestIterator_SetsResourceType_SingleRecord(t *testing.T) {
	// Create mock single record response (not a pagination envelope)
	response := Record{
		"id":     float64(42),
		"name":   "single-item",
		"status": "active",
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

	// Verify resource type is set
	resourceType, ok := records[0][ResourceTypeKey]
	if !ok {
		t.Error("Record missing @resourceType key")
	} else if resourceType != "Item" {
		t.Errorf("Record has wrong resource type: expected 'Item', got '%v'", resourceType)
	}
}

// Test resource type persists across multiple pages
func TestIterator_SetsResourceType_MultiplePages(t *testing.T) {
	page1 := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "user1"},
			map[string]any{"id": float64(2), "name": "user2"},
		},
		"count":    float64(4),
		"next":     "https://test.example.com:443/api/v1/users/?page=2",
		"previous": nil,
	}

	page2 := Record{
		"results": []any{
			map[string]any{"id": float64(3), "name": "user3"},
			map[string]any{"id": float64(4), "name": "user4"},
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

	// Get page 1
	records1, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error on page 1, got: %v", err)
	}

	if len(records1) != 2 {
		t.Fatalf("Expected 2 records on page 1, got %d", len(records1))
	}

	for i, record := range records1 {
		resourceType, ok := record[ResourceTypeKey]
		if !ok {
			t.Errorf("Page 1, Record %d missing @resourceType key", i)
		} else if resourceType != "User" {
			t.Errorf("Page 1, Record %d has wrong resource type: expected 'User', got '%v'", i, resourceType)
		}
	}

	// Get page 2
	records2, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error on page 2, got: %v", err)
	}

	if len(records2) != 2 {
		t.Fatalf("Expected 2 records on page 2, got %d", len(records2))
	}

	for i, record := range records2 {
		resourceType, ok := record[ResourceTypeKey]
		if !ok {
			t.Errorf("Page 2, Record %d missing @resourceType key", i)
		} else if resourceType != "User" {
			t.Errorf("Page 2, Record %d has wrong resource type: expected 'User', got '%v'", i, resourceType)
		}
	}
}

// Test resource type with All() method
func TestIterator_SetsResourceType_All(t *testing.T) {
	page1 := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "quota1"},
		},
		"count":    float64(2),
		"next":     "https://test.example.com:443/api/v1/quotas/?page=2",
		"previous": nil,
	}

	page2 := Record{
		"results": []any{
			map[string]any{"id": float64(2), "name": "quota2"},
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

	// Use All() to fetch all pages at once
	allRecords, err := iter.All()
	if err != nil {
		t.Fatalf("Expected no error from All(), got: %v", err)
	}

	if len(allRecords) != 2 {
		t.Fatalf("Expected 2 total records, got %d", len(allRecords))
	}

	// Verify resource type is set on all records from All()
	for i, record := range allRecords {
		resourceType, ok := record[ResourceTypeKey]
		if !ok {
			t.Errorf("Record %d from All() missing @resourceType key", i)
		} else if resourceType != "Quota" {
			t.Errorf("Record %d from All() has wrong resource type: expected 'Quota', got '%v'", i, resourceType)
		}
	}
}

// Test resource type with Reset() method
func TestIterator_SetsResourceType_Reset(t *testing.T) {
	response := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "policy1"},
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

	// First fetch
	_, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error on first Next(), got: %v", err)
	}

	// Reset and fetch again
	records, err := iter.Reset()
	if err != nil {
		t.Fatalf("Expected no error from Reset(), got: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record after Reset(), got %d", len(records))
	}

	// Verify resource type is set after reset
	resourceType, ok := records[0][ResourceTypeKey]
	if !ok {
		t.Error("Record after Reset() missing @resourceType key")
	} else if resourceType != "Policy" {
		t.Errorf("Record after Reset() has wrong resource type: expected 'Policy', got '%v'", resourceType)
	}
}

// Test that iterator with Dummy resource doesn't set resource type
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

	// Create a Dummy resource
	dummyResource := NewVastResource("/dummy", "Dummy", mockRest, 0, nil)

	// Create iterator
	iter := NewResourceIterator(context.Background(), dummyResource, Params{}, 10)

	// Get first page
	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	// Verify that @resourceType is NOT set for Dummy resources
	for i, record := range records {
		if _, ok := record[ResourceTypeKey]; ok {
			t.Errorf("Record %d should not have @resourceType for Dummy resource", i)
		}
	}
}
