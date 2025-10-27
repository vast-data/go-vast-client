package core

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// mockSessionForVastResourceIterator is a mock session for testing VastResource iterator methods
type mockSessionForVastResourceIterator struct {
	pageSize  int
	responses map[string]Renderable
}

func (m *mockSessionForVastResourceIterator) Get(ctx context.Context, url string, params Params, headers []http.Header) (Renderable, error) {
	if response, ok := m.responses[url]; ok {
		return response, nil
	}
	return RecordSet{}, fmt.Errorf("URL not found in mock: %s (available: %v)", url, m.responses)
}

func (m *mockSessionForVastResourceIterator) Post(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSessionForVastResourceIterator) Put(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSessionForVastResourceIterator) Patch(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSessionForVastResourceIterator) Delete(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSessionForVastResourceIterator) GetConfig() *VMSConfig {
	return &VMSConfig{
		Host:       "test.example.com",
		Port:       443,
		ApiVersion: "v1",
		PageSize:   m.pageSize,
	}
}

func (m *mockSessionForVastResourceIterator) GetAuthenticator() Authenticator {
	return nil
}

// Test GetIterator and GetIteratorWithContext methods on VastResource
func TestVastResource_GetIterator(t *testing.T) {
	// Create a mock session
	session := &mockSessionForVastResourceIterator{
		pageSize:  100,
		responses: map[string]Renderable{},
	}

	// Create a DummyRest with context
	ctx := context.Background()
	rest := &DummyRest{
		ctx:         ctx,
		Session:     session,
		resourceMap: make(map[string]VastResourceAPIWithContext),
	}

	// Create a VastResource (note: resource path is relative, API version will be prepended)
	resource := NewVastResource("/test", "TestResource", rest, 0, nil)

	// Test GetIterator (should use rest context)
	iter := resource.GetIterator(Params{"key": "value"}, 50)
	if iter == nil {
		t.Fatal("Expected iterator, got nil")
	}

	// Verify iterator was created with correct page size
	if iter.PageSize() != 50 {
		t.Errorf("Expected page size 50, got %d", iter.PageSize())
	}
}

func TestVastResource_GetIteratorWithContext(t *testing.T) {
	session := &mockSessionForVastResourceIterator{
		pageSize:  0, // Test default
		responses: map[string]Renderable{},
	}

	rest := &DummyRest{
		ctx:         context.Background(),
		Session:     session,
		resourceMap: make(map[string]VastResourceAPIWithContext),
	}

	resource := NewVastResource("/test", "TestResource", rest, 0, nil)

	// Test with custom context
	ctx := context.WithValue(context.Background(), "test", "value")
	iter := resource.GetIteratorWithContext(ctx, Params{}, 0)

	if iter == nil {
		t.Fatal("Expected iterator, got nil")
	}

	// Verify default page size from config (0)
	if iter.PageSize() != 0 {
		t.Errorf("Expected page size 0, got %d", iter.PageSize())
	}
}

func TestVastResource_GetIterator_DefaultPageSize(t *testing.T) {
	// Test that GetIterator uses session's configured PageSize when 0 is passed
	session := &mockSessionForVastResourceIterator{
		pageSize:  250,
		responses: map[string]Renderable{},
	}

	rest := &DummyRest{
		ctx:         context.Background(),
		Session:     session,
		resourceMap: make(map[string]VastResourceAPIWithContext),
	}

	resource := NewVastResource("/test", "TestResource", rest, 0, nil)

	// Pass 0 to use config default
	iter := resource.GetIterator(Params{}, 0)

	if iter.PageSize() != 250 {
		t.Errorf("Expected page size from config (250), got %d", iter.PageSize())
	}
}

func TestVastResource_ListUsesIterator(t *testing.T) {
	// Test that List() internally uses GetIterator().All()
	records := []any{
		map[string]any{"id": float64(1), "name": "item1"},
		map[string]any{"id": float64(2), "name": "item2"},
	}

	response := Record{
		"results":  records,
		"count":    float64(2),
		"next":     nil,
		"previous": nil,
	}

	session := &mockSessionForVastResourceIterator{
		pageSize: 10,
		responses: map[string]Renderable{
			"https://test.example.com:443/api/v1/test/?page_size=10": response,
		},
	}

	rest := &DummyRest{
		ctx:         context.Background(),
		Session:     session,
		resourceMap: make(map[string]VastResourceAPIWithContext),
	}

	resource := NewVastResource("/test", "TestResource", rest, NewResourceOps(L), nil)

	// Call List - should use iterator internally
	result, err := resource.ListWithContext(context.Background(), Params{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 records, got %d", len(result))
	}
}
