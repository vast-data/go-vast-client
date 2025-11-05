package core

import (
	"testing"
)

// TestExtraMethodRegistry verifies that the registry works correctly
func TestExtraMethodRegistry(t *testing.T) {
	// Clear the registry for this test
	ExtraMethodRegistry = make(map[string]map[string]ExtraMethodMetadata)

	// Register some test methods
	RegisterExtraMethod(
		"apitokens",
		"ApiTokenRevoke_PATCH",
		"PATCH",
		"/apitokens/{id}/revoke/",
		"Revoke API Token",
	)

	RegisterExtraMethod(
		"users",
		"UserQuery_GET",
		"GET",
		"/users/query/",
		"Query Users",
	)

	RegisterExtraMethod(
		"cluster",
		"ClusterUpgrade_PATCH",
		"PATCH",
		"/cluster/{guid}/upgrade/",
		"Upgrade Cluster",
	)

	// Test GetExtraMethodMetadata
	t.Run("GetExtraMethodMetadata", func(t *testing.T) {
		tests := []struct {
			resourceType string
			methodName   string
			wantFound    bool
			wantPath     string
			wantVerb     string
		}{
			{
				resourceType: "apitokens",
				methodName:   "ApiTokenRevoke_PATCH",
				wantFound:    true,
				wantPath:     "/apitokens/{id}/revoke/",
				wantVerb:     "PATCH",
			},
			{
				resourceType: "users",
				methodName:   "UserQuery_GET",
				wantFound:    true,
				wantPath:     "/users/query/",
				wantVerb:     "GET",
			},
			{
				resourceType: "cluster",
				methodName:   "ClusterUpgrade_PATCH",
				wantFound:    true,
				wantPath:     "/cluster/{guid}/upgrade/",
				wantVerb:     "PATCH",
			},
			{
				resourceType: "nonexistent",
				methodName:   "Foo_GET",
				wantFound:    false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.methodName, func(t *testing.T) {
				metadata, found := GetExtraMethodMetadata(tt.resourceType, tt.methodName)

				if found != tt.wantFound {
					t.Errorf("GetExtraMethodMetadata() found = %v, want %v", found, tt.wantFound)
				}

				if found {
					if metadata.URLPath != tt.wantPath {
						t.Errorf("URLPath = %v, want %v", metadata.URLPath, tt.wantPath)
					}
					if metadata.HTTPVerb != tt.wantVerb {
						t.Errorf("HTTPVerb = %v, want %v", metadata.HTTPVerb, tt.wantVerb)
					}
				}
			})
		}
	})

	// Test GetAllExtraMethodsForResource
	t.Run("GetAllExtraMethodsForResource", func(t *testing.T) {
		methods := GetAllExtraMethodsForResource("apitokens")
		if len(methods) != 1 {
			t.Errorf("Expected 1 method for apitokens, got %d", len(methods))
		}
		if len(methods) > 0 {
			if methods[0].MethodName != "ApiTokenRevoke_PATCH" {
				t.Errorf("Expected ApiTokenRevoke_PATCH, got %s", methods[0].MethodName)
			}
		}

		// Test non-existent resource
		methods = GetAllExtraMethodsForResource("nonexistent")
		if methods != nil {
			t.Errorf("Expected nil for nonexistent resource, got %v", methods)
		}
	})
}

// MockApiToken is a test mock for testing extra method discovery
type MockApiToken struct{}

func (m *MockApiToken) GetResourceType() string {
	return "apitokens"
}

func (m *MockApiToken) GetResourcePath() string {
	return "/apitokens/"
}

// ApiTokenRevoke_PATCH is an extra method that would be called by the TUI
func (m *MockApiToken) ApiTokenRevoke_PATCH(id any, body Params) (Record, error) {
	return Record{}, nil
}

// TestDiscoverExtraMethodsWithRegistry tests discovery with registered metadata
func TestDiscoverExtraMethodsWithRegistry(t *testing.T) {
	// Clear and populate registry
	ExtraMethodRegistry = make(map[string]map[string]ExtraMethodMetadata)

	RegisterExtraMethod(
		"apitokens",
		"ApiTokenRevoke_PATCH",
		"PATCH",
		"/apitokens/{id}/revoke/",
		"Revoke API Token",
	)

	// Discover methods
	// NOTE: We now use registry-based discovery instead of reflection!
	// The actual method implementation on the mock is no longer required for discovery.
	// Discovery is now 100% based on the metadata registry.
	mock := &MockApiToken{}
	methods := DiscoverExtraMethodsFromResource(mock)

	t.Logf("Discovered %d methods", len(methods))
	for _, method := range methods {
		t.Logf("  Method: %s, Verb: %s, Path: %s", method.Name, method.HTTPVerb, method.Path)
	}

	// Verify we found the revoke method with correct path from registry
	if len(methods) != 1 {
		t.Errorf("Expected 1 method, got %d", len(methods))
	}

	found := false
	for _, method := range methods {
		if method.Name == "ApiTokenRevoke_PATCH" {
			found = true
			if method.Path != "/apitokens/{id}/revoke/" {
				t.Errorf("Expected path /apitokens/{id}/revoke/, got %s", method.Path)
			}
			if method.HTTPVerb != "PATCH" {
				t.Errorf("Expected verb PATCH, got %s", method.HTTPVerb)
			}
		}
	}

	if !found {
		t.Error("ApiTokenRevoke_PATCH method not discovered")
	}
}
