package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockResourceForIterator is a mock implementation of VastResourceAPIWithContext for testing
type mockResourceForIterator struct {
	*VastResource
	mockSession *mockSessionForIterator
}

func (m *mockResourceForIterator) Session() RESTSession {
	return m.mockSession
}

// mockSessionForIterator is a mock RESTSession for testing
type mockSessionForIterator struct {
	responses map[string]Renderable
	getCount  int
}

func (m *mockSessionForIterator) Get(ctx context.Context, url string, params Params, headers []http.Header) (Renderable, error) {
	m.getCount++

	if response, ok := m.responses[url]; ok {
		return response, nil
	}

	return RecordSet{}, nil
}

func (m *mockSessionForIterator) Post(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSessionForIterator) Put(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSessionForIterator) Patch(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSessionForIterator) Delete(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSessionForIterator) GetConfig() *VMSConfig {
	return &VMSConfig{
		Host:       "test.example.com",
		Port:       443,
		ApiVersion: "v1",
	}
}

func (m *mockSessionForIterator) GetAuthenticator() Authenticator {
	return nil
}

// Test basic iteration with paginated response
func TestIterator_PaginatedResponse(t *testing.T) {
	// Create mock responses
	page1 := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "item1"},
			map[string]any{"id": float64(2), "name": "item2"},
		},
		"count":    float64(4),
		"next":     "https://test.example.com:443/api/v1/resources/?page=2",
		"previous": nil,
	}

	page2 := Record{
		"results": []any{
			map[string]any{"id": float64(3), "name": "item3"},
			map[string]any{"id": float64(4), "name": "item4"},
		},
		"count":    float64(4),
		"next":     nil,
		"previous": "https://test.example.com:443/api/v1/resources/?page=1",
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/resources/?page_size=2": page1,
			"https://test.example.com:443/api/v1/resources/?page=2":      page2,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	mockResource := &mockResourceForIterator{
		VastResource: &VastResource{
			resourcePath: "resources",
			resourceType: "TestResource",
			Rest:         mockRest,
		},
		mockSession: mockSession,
	}

	// Create iterator
	iter := NewResourceIterator(context.Background(), mockResource, Params{}, 2)

	// First page
	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected first page, got error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Expected 2 items in first page, got %d", len(records))
	}

	if iter.Count() != 4 {
		t.Errorf("Expected total count of 4, got %d", iter.Count())
	}

	if !iter.HasNext() {
		t.Error("Expected HasNext to be true")
	}

	if iter.HasPrevious() {
		t.Error("Expected HasPrevious to be false on first page")
	}

	// Second page
	records, err = iter.Next()
	if err != nil {
		t.Fatalf("Expected second page, got error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Expected 2 items in second page, got %d", len(records))
	}

	if iter.HasNext() {
		t.Error("Expected HasNext to be false on last page")
	}

	if !iter.HasPrevious() {
		t.Error("Expected HasPrevious to be true on second page")
	}

	// No more pages
	records, err = iter.Next()
	if err != nil {
		t.Error("Unexpected error on last Next()")
	}
	if len(records) != 0 {
		t.Error("Expected no more records")
	}
}

// Test iteration with non-paginated response
func TestIterator_NonPaginatedResponse(t *testing.T) {
	// Create mock non-paginated response (flat array)
	flatResponse := RecordSet{
		{"id": float64(1), "name": "item1"},
		{"id": float64(2), "name": "item2"},
		{"id": float64(3), "name": "item3"},
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/resources/?page_size=10": flatResponse,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	mockResource := &mockResourceForIterator{
		VastResource: &VastResource{
			resourcePath: "resources",
			resourceType: "TestResource",
			Rest:         mockRest,
		},
		mockSession: mockSession,
	}

	// Create iterator
	iter := NewResourceIterator(context.Background(), mockResource, Params{}, 10)

	// First (and only) page
	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected first page, got error: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("Expected 3 items, got %d", len(records))
	}

	if iter.HasNext() {
		t.Error("Expected HasNext to be false for non-paginated response")
	}

	// No more pages
	records, err = iter.Next()
	if err != nil {
		t.Error("Unexpected error")
	}
	if len(records) != 0 {
		t.Error("Expected no more records")
	}
}

// Test All() method
func TestIterator_All(t *testing.T) {
	// Create mock responses with multiple pages
	page1 := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "item1"},
			map[string]any{"id": float64(2), "name": "item2"},
		},
		"count":    float64(6),
		"next":     "https://test.example.com:443/api/v1/resources/?page=2",
		"previous": nil,
	}

	page2 := Record{
		"results": []any{
			map[string]any{"id": float64(3), "name": "item3"},
			map[string]any{"id": float64(4), "name": "item4"},
		},
		"count":    float64(6),
		"next":     "https://test.example.com:443/api/v1/resources/?page=3",
		"previous": "https://test.example.com:443/api/v1/resources/?page=1",
	}

	page3 := Record{
		"results": []any{
			map[string]any{"id": float64(5), "name": "item5"},
			map[string]any{"id": float64(6), "name": "item6"},
		},
		"count":    float64(6),
		"next":     nil,
		"previous": "https://test.example.com:443/api/v1/resources/?page=2",
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/resources/?page_size=2": page1,
			"https://test.example.com:443/api/v1/resources/?page=2":      page2,
			"https://test.example.com:443/api/v1/resources/?page=3":      page3,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	mockResource := &mockResourceForIterator{
		VastResource: &VastResource{
			resourcePath: "resources",
			resourceType: "TestResource",
			Rest:         mockRest,
		},
		mockSession: mockSession,
	}

	// Create iterator
	iter := NewResourceIterator(context.Background(), mockResource, Params{}, 2)

	// Fetch all pages
	allRecords, err := iter.All()
	if err != nil {
		t.Fatalf("Expected no error from All(), got: %v", err)
	}

	if len(allRecords) != 6 {
		t.Errorf("Expected 6 total records, got %d", len(allRecords))
	}

	// Verify all records are present
	for i := 0; i < 6; i++ {
		expectedID := i + 1
		if int(allRecords[i]["id"].(float64)) != expectedID {
			t.Errorf("Expected record %d to have id %d, got %v", i, expectedID, allRecords[i]["id"])
		}
	}
}

// Test Reset() method
func TestIterator_Reset(t *testing.T) {
	page1 := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "item1"},
			map[string]any{"id": float64(2), "name": "item2"},
		},
		"count":    float64(4),
		"next":     "https://test.example.com:443/api/v1/resources/?page=2",
		"previous": nil,
	}

	page2 := Record{
		"results": []any{
			map[string]any{"id": float64(3), "name": "item3"},
			map[string]any{"id": float64(4), "name": "item4"},
		},
		"count":    float64(4),
		"next":     nil,
		"previous": "https://test.example.com:443/api/v1/resources/?page=1",
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/resources/?page_size=2": page1,
			"https://test.example.com:443/api/v1/resources/?page=2":      page2,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	mockResource := &mockResourceForIterator{
		VastResource: &VastResource{
			resourcePath: "resources",
			resourceType: "TestResource",
			Rest:         mockRest,
		},
		mockSession: mockSession,
	}

	// Create iterator
	iter := NewResourceIterator(context.Background(), mockResource, Params{}, 2)

	// Iterate through all pages
	pageCount := 0
	for {
		records, err := iter.Next()
		if err != nil || len(records) == 0 {
			break
		}
		pageCount++
	}

	if pageCount != 2 {
		t.Errorf("Expected 2 pages, got %d", pageCount)
	}

	// Reset iterator
	records, err := iter.Reset()
	if err != nil {
		t.Fatalf("Expected no error from Reset(), got: %v", err)
	}
	if len(records) == 0 {
		t.Error("Expected records from Reset()")
	}

	// Should be able to iterate again
	pageCount = 0
	for {
		records, err := iter.Next()
		if err != nil || len(records) == 0 {
			break
		}
		pageCount++
	}

	if pageCount != 1 { // Already got first page from Reset()
		t.Errorf("Expected 1 more page after reset, got %d", pageCount)
	}
}

// Test PageSize() method
func TestIterator_PageSize(t *testing.T) {
	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	mockResource := &mockResourceForIterator{
		VastResource: &VastResource{
			resourcePath: "resources",
			resourceType: "TestResource",
			Rest:         mockRest,
		},
		mockSession: mockSession,
	}

	// Test with explicit page size
	iter := NewResourceIterator(context.Background(), mockResource, Params{}, 50)
	if iter.PageSize() != 50 {
		t.Errorf("Expected page size 50, got %d", iter.PageSize())
	}

	// Test with default page size (0 should remain 0 - no page_size param)
	iter = NewResourceIterator(context.Background(), mockResource, Params{}, 0)
	if iter.PageSize() != 0 {
		t.Errorf("Expected default page size 0, got %d", iter.PageSize())
	}

	// Test with negative page size (should default to 0 - no page_size param)
	iter = NewResourceIterator(context.Background(), mockResource, Params{}, -1)
	if iter.PageSize() != 0 {
		t.Errorf("Expected default page size 0 for negative input, got %d", iter.PageSize())
	}
}

// Test Previous() navigation
func TestIterator_Previous(t *testing.T) {
	page1 := Record{
		"results": []any{
			map[string]any{"id": float64(1), "name": "item1"},
			map[string]any{"id": float64(2), "name": "item2"},
		},
		"count":    float64(4),
		"next":     "https://test.example.com:443/api/v1/resources/?page=2",
		"previous": nil,
	}

	page2 := Record{
		"results": []any{
			map[string]any{"id": float64(3), "name": "item3"},
			map[string]any{"id": float64(4), "name": "item4"},
		},
		"count":    float64(4),
		"next":     nil,
		"previous": "https://test.example.com:443/api/v1/resources/?page=1",
	}

	// For previous navigation, page1 is returned again
	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/resources/?page_size=2": page1,
			"https://test.example.com:443/api/v1/resources/?page=2":      page2,
			"https://test.example.com:443/api/v1/resources/?page=1":      page1,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	mockResource := &mockResourceForIterator{
		VastResource: &VastResource{
			resourcePath: "resources",
			resourceType: "TestResource",
			Rest:         mockRest,
		},
		mockSession: mockSession,
	}

	// Create iterator
	iter := NewResourceIterator(context.Background(), mockResource, Params{}, 2)

	// Move to page 1
	records, err := iter.Next()
	if err != nil {
		t.Fatalf("Expected first page, got error: %v", err)
	}

	// Move to page 2
	records, err = iter.Next()
	if err != nil {
		t.Fatalf("Expected second page, got error: %v", err)
	}

	// Check current page has correct IDs
	if len(records) != 2 {
		t.Errorf("Expected 2 items, got %d", len(records))
	}
	if int(records[0]["id"].(float64)) != 3 {
		t.Errorf("Expected first item ID to be 3, got %v", records[0]["id"])
	}

	// Move back to page 1
	records, err = iter.Previous()
	if err != nil {
		t.Fatalf("Expected to move to previous page, got error: %v", err)
	}

	// Verify we're back on page 1
	if len(records) != 2 {
		t.Errorf("Expected 2 items, got %d", len(records))
	}
	if int(records[0]["id"].(float64)) != 1 {
		t.Errorf("Expected first item ID to be 1, got %v", records[0]["id"])
	}

	// Try to go before first page (should return empty)
	records, err = iter.Previous()
	if err != nil {
		t.Error("Expected no error when going before first page")
	}
	if len(records) != 0 {
		t.Error("Expected empty records before first page")
	}
}

// Benchmark iterator performance
func BenchmarkIterator_Next(b *testing.B) {
	// Create a simple response
	response := Record{
		"results": []any{
			map[string]any{"id": 1, "name": "item1"},
			map[string]any{"id": 2, "name": "item2"},
		},
		"count":    2,
		"next":     nil,
		"previous": nil,
	}

	mockSession := &mockSessionForIterator{
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/resources/?page_size=10": response,
		},
	}

	mockRest := &DummyRest{
		ctx:     context.Background(),
		Session: mockSession,
	}

	mockResource := &mockResourceForIterator{
		VastResource: &VastResource{
			resourcePath: "resources",
			resourceType: "TestResource",
			Rest:         mockRest,
		},
		mockSession: mockSession,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		iter := NewResourceIterator(context.Background(), mockResource, Params{}, 10)
		iter.Next()
	}
}

// Test with real HTTP server (integration-style test)
func TestIterator_WithHTTPServer(t *testing.T) {
	// Note: This test demonstrates the structure but won't run without proper session setup
	// It's included to show how integration tests could be structured
	t.Skip("Integration test - requires full session setup")

	// Create a test HTTP server
	var testServer *httptest.Server
	pageNum := 0
	testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageNum++

		if pageNum == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{
				"results": [{"id": 1}, {"id": 2}],
				"count": 4,
				"next": "%s/page2",
				"previous": null
			}`, testServer.URL)
		} else if r.URL.Path == "/page2" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{
				"results": [{"id": 3}, {"id": 4}],
				"count": 4,
				"next": null,
				"previous": "%s"
			}`, testServer.URL)
		}
	}))
	defer testServer.Close()
}

// Test String() method
func TestIterator_String(t *testing.T) {
	// Create a simple iterator with mock data
	records := RecordSet{
		{"id": float64(1), "name": "item1"},
		{"id": float64(2), "name": "item2"},
		{"id": float64(3), "name": "item3"},
	}

	nextURL := "https://api.example.com/resources?page=2"
	prevURL := "https://api.example.com/resources?page=1"

	// Test uninitialized state
	iter := &ResourceIterator{
		pageSize:    50,
		totalCount:  -1,
		initialized: false,
	}

	str := iter.String()
	if !strings.Contains(str, "ResourceIterator") {
		t.Error("Expected String() to contain 'ResourceIterator'")
	}
	if !strings.Contains(str, "Initialized:   false") {
		t.Error("Expected String() to show uninitialized state")
	}
	if !strings.Contains(str, "Current:       []") {
		t.Error("Expected String() to show empty current")
	}

	// Test initialized state with data
	iter.initialized = true
	iter.currentPage = 1
	iter.current = records
	iter.totalCount = 150
	iter.nextURL = &nextURL
	iter.previousURL = &prevURL

	str = iter.String()
	if !strings.Contains(str, "Initialized:   true") {
		t.Error("Expected String() to show initialized state")
	}
	if !strings.Contains(str, "... (3 items)") {
		t.Error("Expected String() to show '... (3 items)'")
	}
	if !strings.Contains(str, "Total Count:   150") {
		t.Error("Expected String() to show total count")
	}
	if !strings.Contains(str, "Next URL:") {
		t.Error("Expected String() to show next URL")
	}
	if !strings.Contains(str, "Previous URL:") && !strings.Contains(str, "<none>") {
		t.Error("Expected String() to show previous URL")
	}

	// Test empty state
	iter.current = RecordSet{}
	str = iter.String()
	if !strings.Contains(str, "Current:       []") {
		t.Error("Expected String() to show empty brackets for empty current")
	}
}
