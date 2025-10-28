package core

import (
	"context"
	"testing"
)

// MockRest implements VastRest for testing
type MockRest struct {
	ctx         context.Context
	session     RESTSession
	resourceMap map[string]VastResourceAPIWithContext
}

func (m *MockRest) GetSession() RESTSession {
	return m.session
}

func (m *MockRest) GetResourceMap() map[string]VastResourceAPIWithContext {
	return m.resourceMap
}

func (m *MockRest) GetCtx() context.Context {
	return m.ctx
}

func (m *MockRest) SetCtx(ctx context.Context) {
	m.ctx = ctx
}

// TestResourceOpsValidation tests that ResourceOps correctly validates operations
// Note: This test was simplified to test the has() method directly since
// checkOperation is now a private method and operation validation happens
// inside the actual CRUD methods (ListWithContext, CreateWithContext, etc.)
func TestResourceOpsValidation(t *testing.T) {
	tests := []struct {
		name        string
		resourceOps ResourceOps
		checkOp     ResourceOps
		expected    bool
	}{
		{
			name:        "Read-only resource has Read",
			resourceOps: NewResourceOps(R),
			checkOp:     R,
			expected:    true,
		},
		{
			name:        "Read-only resource does not have Create",
			resourceOps: NewResourceOps(R),
			checkOp:     C,
			expected:    false,
		},
		{
			name:        "Read-only resource does not have Update",
			resourceOps: NewResourceOps(R),
			checkOp:     U,
			expected:    false,
		},
		{
			name:        "Read-only resource does not have Delete",
			resourceOps: NewResourceOps(R),
			checkOp:     D,
			expected:    false,
		},
		{
			name:        "CRUD resource has Create",
			resourceOps: NewResourceOps(C, R, U, D),
			checkOp:     C,
			expected:    true,
		},
		{
			name:        "CRUD resource has Read",
			resourceOps: NewResourceOps(C, R, U, D),
			checkOp:     R,
			expected:    true,
		},
		{
			name:        "CRUD resource has Update",
			resourceOps: NewResourceOps(C, R, U, D),
			checkOp:     U,
			expected:    true,
		},
		{
			name:        "CRUD resource has Delete",
			resourceOps: NewResourceOps(C, R, U, D),
			checkOp:     D,
			expected:    true,
		},
		{
			name:        "RUD resource has Update",
			resourceOps: NewResourceOps(R, U, D),
			checkOp:     U,
			expected:    true,
		},
		{
			name:        "RUD resource does not have Create",
			resourceOps: NewResourceOps(R, U, D),
			checkOp:     C,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.resourceOps.has(tt.checkOp)
			if result != tt.expected {
				t.Errorf("Expected has(%v) = %v, got %v", tt.checkOp, tt.expected, result)
			}
		})
	}
}

func TestResourceOpsString(t *testing.T) {
	tests := []struct {
		name     string
		ops      ResourceOps
		expected string
	}{
		{"No operations", NewResourceOps(), "-"},
		{"Create only", NewResourceOps(C), "C"},
		{"Read only", NewResourceOps(R), "R"},
		{"Update only", NewResourceOps(U), "U"},
		{"Delete only", NewResourceOps(D), "D"},
		{"CRUD", NewResourceOps(C, R, U, D), "CRUD"},
		{"RUD", NewResourceOps(R, U, D), "RUD"},
		{"CR", NewResourceOps(C, R), "CR"},
		{"CU", NewResourceOps(C, U), "CU"},
		{"CD", NewResourceOps(C, D), "CD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ops.String()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestResourceOpsHas(t *testing.T) {
	ops := NewResourceOps(R, U, D) // RUD resource

	tests := []struct {
		name     string
		flag     ResourceOps
		expected bool
	}{
		{"Has Read", R, true},
		{"Has Update", U, true},
		{"Has Delete", D, true},
		{"Does not have Create", C, false},
		{"Does not have List", L, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ops.has(tt.flag)
			if result != tt.expected {
				t.Errorf("Expected has(%v) = %v, got %v", tt.flag, tt.expected, result)
			}
		})
	}
}

// MockResourceWithExtraMethods is a test resource with extra methods
type MockResourceWithExtraMethods struct {
	*VastResource
}

func (m *MockResourceWithExtraMethods) UserAccessKeys_POST(params Params) (Record, error) {
	return nil, nil
}

func (m *MockResourceWithExtraMethods) UserAccessKeys_DELETE(id any, params Params) (Record, error) {
	return Record{}, nil
}

func (m *MockResourceWithExtraMethods) UserQuery_GET(params Params) (RecordSet, error) {
	return nil, nil
}

func TestGetCRUDHintsFromResource(t *testing.T) {
	// Create a test resource with specific operations
	vr := &VastResource{
		resourceType: "users",
		resourcePath: "/users/",
		resourceOps:  NewResourceOps(C, L, R, U, D),
	}

	// Test with direct VastResource
	hints := GetCRUDHintsFromResource(vr)
	if hints != NewResourceOps(C, L, R, U, D) {
		t.Errorf("Expected CLRUD, got %s", hints.String())
	}

	// Test with wrapped resource (like actual API resources)
	wrapped := &MockResourceWithExtraMethods{VastResource: vr}
	hints = GetCRUDHintsFromResource(wrapped)
	if hints != NewResourceOps(C, L, R, U, D) {
		t.Errorf("Expected CLRUD for wrapped resource, got %s", hints.String())
	}

	// Test with read-only resource
	roResource := &VastResource{
		resourceType: "readonly",
		resourcePath: "/readonly/",
		resourceOps:  NewResourceOps(L, R),
	}
	hints = GetCRUDHintsFromResource(roResource)
	if hints != NewResourceOps(L, R) {
		t.Errorf("Expected LR, got %s", hints.String())
	}
}

func TestDiscoverExtraMethodsFromResource(t *testing.T) {
	vr := &VastResource{
		resourceType: "user",
		resourcePath: "/users/",
		resourceOps:  NewResourceOps(C, L, R, U, D),
	}

	mock := &MockResourceWithExtraMethods{VastResource: vr}

	extraMethods := DiscoverExtraMethodsFromResource(mock)

	// Should find 3 extra methods: UserAccessKeys_POST, UserAccessKeys_DELETE, UserQuery_GET
	if len(extraMethods) != 3 {
		t.Fatalf("Expected 3 extra methods, got %d", len(extraMethods))
	}

	// Verify the methods were discovered correctly
	methodNames := make(map[string]bool)
	for _, method := range extraMethods {
		methodNames[method.Name] = true

		// Verify each method has the expected fields
		if method.HTTPVerb == "" {
			t.Errorf("Method %s has empty HTTPVerb", method.Name)
		}
		if method.Path == "" {
			t.Errorf("Method %s has empty Path", method.Name)
		}
	}

	// Check expected methods are present
	expectedMethods := []string{"UserAccessKeys_POST", "UserAccessKeys_DELETE", "UserQuery_GET"}
	for _, expected := range expectedMethods {
		if !methodNames[expected] {
			t.Errorf("Expected method %s not found", expected)
		}
	}
}

func TestInferPathFromMethodName(t *testing.T) {
	tests := []struct {
		name         string
		methodName   string
		resourceType string
		resourcePath string
		expected     string
	}{
		{
			name:         "User access keys",
			methodName:   "UserAccessKeys",
			resourceType: "user",
			resourcePath: "/users/",
			expected:     "/users/access_keys/",
		},
		{
			name:         "User query",
			methodName:   "UserQuery",
			resourceType: "user",
			resourcePath: "/users/",
			expected:     "/users/query/",
		},
		{
			name:         "Host discovered hosts",
			methodName:   "HostDiscoveredHosts",
			resourceType: "host",
			resourcePath: "/hosts/",
			expected:     "/hosts/discovered_hosts/",
		},
		{
			name:         "Empty method name (standard CRUD)",
			methodName:   "User",
			resourceType: "user",
			resourcePath: "/users/",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferPathFromMethodName(tt.methodName, tt.resourceType, tt.resourcePath)
			if result != tt.expected {
				t.Errorf("Expected path '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
