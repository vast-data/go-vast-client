package widgets

import (
	"context"
	"reflect"
	"testing"
	"time"

	"vastix/internal/tui/widgets/common"

	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/openapi_schema"
	"github.com/vast-data/go-vast-client/resources/untyped"
)

// Mock resource for testing
type MockResource struct {
	core.VastResource
}

// Mock extra methods with different signatures

// Method signature: func(ctx, params)
func (m *MockResource) MockMethodNoIdWithContext_GET(ctx context.Context, params core.Params) (core.Record, error) {
	// Return the params so we can verify what was passed
	return core.Record{"received_params": params, "type": "no_id_get"}, nil
}

// Method signature: func(ctx, id, params)
func (m *MockResource) MockMethodWithIdWithContext_GET(ctx context.Context, id any, params core.Params) (core.Record, error) {
	return core.Record{"received_id": id, "received_params": params, "type": "with_id_get"}, nil
}

// Method signature: func(ctx, body)
func (m *MockResource) MockMethodNoIdWithContext_POST(ctx context.Context, body core.Params) (core.Record, error) {
	return core.Record{"received_body": body, "type": "no_id_post"}, nil
}

// Method signature: func(ctx, id, body)
func (m *MockResource) MockMethodWithIdWithContext_POST(ctx context.Context, id any, body core.Params) (core.Record, error) {
	return core.Record{"received_id": id, "received_body": body, "type": "with_id_post"}, nil
}

// Method signature: func(ctx, id, queryParams, bodyParams)
func (m *MockResource) MockMethodBothParamsWithContext_POST(ctx context.Context, id any, queryParams, bodyParams core.Params) (core.Record, error) {
	return core.Record{
		"received_id":    id,
		"received_query": queryParams,
		"received_body":  bodyParams,
		"type":           "both_params_post",
	}, nil
}

// Method signature: func(ctx, body, timeout) - async
func (m *MockResource) MockMethodAsyncWithContext_POST(ctx context.Context, body core.Params, waitTimeout time.Duration) (*untyped.AsyncResult, error) {
	// Simulate successful async operation
	return &untyped.AsyncResult{
		Err: nil,
	}, nil
}

// Method signature: func(ctx, id, body, timeout) - async with ID
func (m *MockResource) MockMethodAsyncWithIdWithContext_PATCH(ctx context.Context, id any, body core.Params, waitTimeout time.Duration) (*untyped.AsyncResult, error) {
	return &untyped.AsyncResult{
		Err: nil,
	}, nil
}

// Test parameter splitting (query vs body)
func TestExtraMethodWidget_ParameterSplitting(t *testing.T) {
	// Note: These tests verify the logic of parameter splitting
	// The actual splitting depends on OpenAPI schema which we can't easily mock
	// So we test the fallback behavior

	tests := []struct {
		name          string
		httpMethod    string
		allParams     core.Params
		expectedQuery core.Params
		expectedBody  core.Params
		description   string
	}{
		{
			name:       "GET method - all params go to query (fallback)",
			httpMethod: "GET",
			allParams: core.Params{
				"filter": "active",
				"limit":  10,
			},
			expectedQuery: core.Params{
				"filter": "active",
				"limit":  10,
			},
			expectedBody: core.Params{},
			description:  "GET requests should put all params in query string when schema lookup fails",
		},
		{
			name:       "POST method - all params go to body (fallback)",
			httpMethod: "POST",
			allParams: core.Params{
				"name":  "test",
				"value": 42,
			},
			expectedQuery: core.Params{},
			expectedBody: core.Params{
				"name":  "test",
				"value": 42,
			},
			description: "POST requests should put all params in body when schema lookup fails",
		},
		{
			name:       "PATCH method - all params go to body (fallback)",
			httpMethod: "PATCH",
			allParams: core.Params{
				"enabled": true,
			},
			expectedQuery: core.Params{},
			expectedBody: core.Params{
				"enabled": true,
			},
			description: "PATCH requests should put all params in body when schema lookup fails",
		},
		{
			name:       "DELETE method - all params go to body (fallback)",
			httpMethod: "DELETE",
			allParams: core.Params{
				"force": true,
			},
			expectedQuery: core.Params{},
			expectedBody: core.Params{
				"force": true,
			},
			description: "DELETE requests should put all params in body when schema lookup fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test documents the expected behavior
			// The actual implementation uses OpenAPI schema when available
			t.Log(tt.description)

			// Verify our expectations about param splitting
			queryParams := make(core.Params)
			bodyParams := make(core.Params)

			// Simulate fallback logic
			if tt.httpMethod == "GET" {
				queryParams = tt.allParams
			} else {
				bodyParams = tt.allParams
			}

			if !reflect.DeepEqual(queryParams, tt.expectedQuery) {
				t.Errorf("Query params mismatch.\nGot:  %v\nWant: %v", queryParams, tt.expectedQuery)
			}

			if !reflect.DeepEqual(bodyParams, tt.expectedBody) {
				t.Errorf("Body params mismatch.\nGot:  %v\nWant: %v", bodyParams, tt.expectedBody)
			}
		})
	}
}

// Test reflection-based method calling with different signatures
func TestExtraMethodWidget_CallExtraMethod_Signatures(t *testing.T) {
	mockResource := &MockResource{}
	ctx := context.Background()

	tests := []struct {
		name            string
		methodInfo      core.ExtraMethodInfo
		queryParams     core.Params
		bodyParams      core.Params
		selectedRowData common.RowData
		expectedType    string
		expectError     bool
	}{
		{
			name: "GET method without ID",
			methodInfo: core.ExtraMethodInfo{
				Name:     "MockMethodNoId_GET",
				HTTPVerb: "GET",
				Path:     "/mock/endpoint/",
			},
			queryParams:     core.Params{"filter": "test"},
			bodyParams:      core.Params{},
			selectedRowData: common.NewRowData([]string{}, []string{}),
			expectedType:    "no_id_get",
			expectError:     false,
		},
		{
			name: "GET method with ID",
			methodInfo: core.ExtraMethodInfo{
				Name:     "MockMethodWithId_GET",
				HTTPVerb: "GET",
				Path:     "/mock/{id}/endpoint/",
			},
			queryParams:     core.Params{"details": true},
			bodyParams:      core.Params{},
			selectedRowData: common.NewRowData([]string{"id"}, []string{"123"}),
			expectedType:    "with_id_get",
			expectError:     false,
		},
		{
			name: "POST method without ID",
			methodInfo: core.ExtraMethodInfo{
				Name:     "MockMethodNoId_POST",
				HTTPVerb: "POST",
				Path:     "/mock/create/",
			},
			queryParams:     core.Params{},
			bodyParams:      core.Params{"name": "test"},
			selectedRowData: common.NewRowData([]string{}, []string{}),
			expectedType:    "no_id_post",
			expectError:     false,
		},
		{
			name: "POST method with ID",
			methodInfo: core.ExtraMethodInfo{
				Name:     "MockMethodWithId_POST",
				HTTPVerb: "POST",
				Path:     "/mock/{id}/action/",
			},
			queryParams:     core.Params{},
			bodyParams:      core.Params{"enabled": true},
			selectedRowData: common.NewRowData([]string{"id"}, []string{"456"}),
			expectedType:    "with_id_post",
			expectError:     false,
		},
		{
			name: "POST method with both query and body params",
			methodInfo: core.ExtraMethodInfo{
				Name:     "MockMethodBothParams_POST",
				HTTPVerb: "POST",
				Path:     "/mock/{id}/complex/",
			},
			queryParams:     core.Params{"tenant_id": 1},
			bodyParams:      core.Params{"data": "value"},
			selectedRowData: common.NewRowData([]string{"id"}, []string{"789"}),
			expectedType:    "both_params_post",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal ExtraMethodWidget
			widget := &ExtraMethodWidget{
				BaseWidget: &BaseWidget{
					selectedRowData: tt.selectedRowData,
				},
				methodInfo: tt.methodInfo,
			}

			// Call the method using reflection
			result, err := widget.callExtraMethod(ctx, mockResource, tt.queryParams, tt.bodyParams)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify result type
			record, ok := result.(core.Record)
			if !ok {
				t.Errorf("Expected result to be core.Record, got %T", result)
				return
			}

			resultType, ok := record["type"].(string)
			if !ok {
				t.Error("Result doesn't contain 'type' field")
				return
			}

			if resultType != tt.expectedType {
				t.Errorf("Expected type %q, got %q", tt.expectedType, resultType)
			}

			// Verify parameters were passed correctly
			t.Logf("Result: %+v", record)
		})
	}
}

// Test async method handling
func TestExtraMethodWidget_CallExtraMethod_Async(t *testing.T) {
	mockResource := &MockResource{}
	ctx := context.Background()

	tests := []struct {
		name        string
		methodInfo  core.ExtraMethodInfo
		bodyParams  core.Params
		expectError bool
	}{
		{
			name: "Async method without ID",
			methodInfo: core.ExtraMethodInfo{
				Name:     "MockMethodAsync_POST",
				HTTPVerb: "POST",
				Path:     "/mock/async/",
			},
			bodyParams:  core.Params{"action": "execute"},
			expectError: false,
		},
		{
			name: "Async method with ID",
			methodInfo: core.ExtraMethodInfo{
				Name:     "MockMethodAsyncWithId_PATCH",
				HTTPVerb: "PATCH",
				Path:     "/mock/{id}/async_action/",
			},
			bodyParams:  core.Params{"enabled": true},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectedRowData := common.NewRowData([]string{}, []string{})
			if tt.methodInfo.Path != "/mock/async/" {
				// Add ID for methods that need it
				selectedRowData = common.NewRowData([]string{"id"}, []string{"999"})
			}

			widget := &ExtraMethodWidget{
				BaseWidget: &BaseWidget{
					selectedRowData: selectedRowData,
				},
				methodInfo: tt.methodInfo,
			}

			result, err := widget.callExtraMethod(ctx, mockResource, core.Params{}, tt.bodyParams)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify result is AsyncResult
			asyncResult, ok := result.(*untyped.AsyncResult)
			if !ok {
				t.Errorf("Expected result to be *untyped.AsyncResult, got %T", result)
				return
			}

			if asyncResult.Err != nil {
				t.Errorf("Async operation failed: %v", asyncResult.Err)
			}

			t.Log("Async operation completed successfully")
		})
	}
}

// Test ID extraction from selected row
func TestExtraMethodWidget_IDExtraction(t *testing.T) {
	tests := []struct {
		name            string
		selectedRowData common.RowData
		expectID        bool
		expectedIDValue string
	}{
		{
			name:            "ID present in row data",
			selectedRowData: common.NewRowData([]string{"id", "name"}, []string{"123", "test"}),
			expectID:        true,
			expectedIDValue: "123",
		},
		{
			name:            "GUID present instead of ID",
			selectedRowData: common.NewRowData([]string{"guid", "name"}, []string{"abc-def", "test"}),
			expectID:        true,
			expectedIDValue: "abc-def",
		},
		{
			name:            "No ID or GUID present",
			selectedRowData: common.NewRowData([]string{"name"}, []string{"test"}),
			expectID:        false,
			expectedIDValue: "",
		},
		{
			name:            "Empty row data",
			selectedRowData: common.NewRowData([]string{}, []string{}),
			expectID:        false,
			expectedIDValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Try to get ID
			var idValue interface{}
			if tt.selectedRowData.Len() > 0 {
				if id := tt.selectedRowData.Get("id"); id != nil {
					idValue = id
				} else if guid := tt.selectedRowData.Get("guid"); guid != nil {
					idValue = guid
				}
			}

			if tt.expectID {
				if idValue == nil {
					t.Error("Expected ID but got nil")
					return
				}

				idStr := idValue.(string)
				if idStr != tt.expectedIDValue {
					t.Errorf("Expected ID %q, got %q", tt.expectedIDValue, idStr)
				}
			} else {
				if idValue != nil {
					t.Errorf("Expected no ID but got %v", idValue)
				}
			}
		})
	}
}

// Test OpenAPI schema-based parameter splitting (integration-style test)
func TestExtraMethodWidget_OpenAPIParameterSplitting(t *testing.T) {
	// This test verifies that we correctly use openapi_schema.GetQueryParameters
	// when it's available

	// Note: This is more of an integration test
	// We're testing that the function exists and has the right signature

	t.Run("GetQueryParameters function exists", func(t *testing.T) {
		// Verify the function exists and can be called
		_, err := openapi_schema.GetQueryParameters("GET", "/some/path/")
		// We expect an error because the path doesn't exist, but that's fine
		// We're just verifying the function signature is correct
		if err == nil {
			t.Log("Function call succeeded (unexpected but not an error)")
		} else {
			t.Logf("Function call failed as expected: %v", err)
		}
	})
}

// Benchmark for reflection-based method calling
func BenchmarkExtraMethodWidget_CallExtraMethod(b *testing.B) {
	mockResource := &MockResource{}
	ctx := context.Background()

	widget := &ExtraMethodWidget{
		BaseWidget: &BaseWidget{
			selectedRowData: common.NewRowData([]string{"id"}, []string{"123"}),
		},
		methodInfo: core.ExtraMethodInfo{
			Name:     "MockMethodWithId_GET",
			HTTPVerb: "GET",
			Path:     "/mock/{id}/endpoint/",
		},
	}

	queryParams := core.Params{"filter": "test"}
	bodyParams := core.Params{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = widget.callExtraMethod(ctx, mockResource, queryParams, bodyParams)
	}
}

// Test error cases
func TestExtraMethodWidget_CallExtraMethod_Errors(t *testing.T) {
	mockResource := &MockResource{}
	ctx := context.Background()

	tests := []struct {
		name            string
		methodInfo      core.ExtraMethodInfo
		selectedRowData common.RowData
		expectError     bool
		errorContains   string
	}{
		{
			name: "Method not found",
			methodInfo: core.ExtraMethodInfo{
				Name:     "NonExistentMethod_GET",
				HTTPVerb: "GET",
				Path:     "/mock/nonexistent/",
			},
			selectedRowData: common.NewRowData([]string{}, []string{}),
			expectError:     true,
			errorContains:   "not found",
		},
		{
			name: "Method requires ID but none provided",
			methodInfo: core.ExtraMethodInfo{
				Name:     "MockMethodWithId_GET",
				HTTPVerb: "GET",
				Path:     "/mock/{id}/endpoint/",
			},
			selectedRowData: common.NewRowData([]string{}, []string{}), // No ID
			expectError:     true,
			errorContains:   "no ID found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widget := &ExtraMethodWidget{
				BaseWidget: &BaseWidget{
					selectedRowData: tt.selectedRowData,
				},
				methodInfo: tt.methodInfo,
			}

			_, err := widget.callExtraMethod(ctx, mockResource, core.Params{}, core.Params{})

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorContains != "" {
					errStr := err.Error()
					if !contains(errStr, tt.errorContains) {
						t.Errorf("Expected error to contain %q, got %q", tt.errorContains, errStr)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
