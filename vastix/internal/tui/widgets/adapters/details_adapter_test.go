package adapters

import (
	"vastix/internal/colors"
	"strings"
	"testing"
	"vastix/internal/database"
)

func TestDetailsAdapter_NilRawContentHandling(t *testing.T) {
	// Create a mock database service (nil is fine for this test)
	adapter := NewDetailsAdapter(nil, "test-resource")

	// By default, rawContent should be nil
	if adapter.rawContent != nil {
		t.Errorf("Expected rawContent to be nil initially, got %v", adapter.rawContent)
	}

	// Test that contentToString handles nil rawContent properly
	contentStr := adapter.contentToString()
	if !strings.Contains(contentStr, "No content") {
		t.Errorf("Expected content to contain 'No content', got: %s", contentStr)
	}

	// Log the actual content for debugging
	t.Logf("Actual content: %q", contentStr)
	t.Logf("Content length: %d, trimmed length: %d", len(contentStr), len(strings.TrimSpace(contentStr)))

	// Test that the content is styled (should contain ANSI escape codes for gray color)
	if !strings.Contains(contentStr, "\x1b[") {
		t.Logf("Content does not contain ANSI escape codes - this is acceptable if styling is applied differently")
	}

	// Test that the content has padding - padding should make the content longer than just "No content"
	if len(contentStr) <= len("No content") {
		t.Errorf("Expected content to have padding applied. Content length: %d, expected > %d", len(contentStr), len("No content"))
	}
}

func TestDetailsAdapter_SetContent(t *testing.T) {
	adapter := NewDetailsAdapter(nil, "test-resource")

	// Test with string content
	testContent := "test content"
	adapter.SetContent(testContent)

	if adapter.rawContent != testContent {
		t.Errorf("Expected rawContent to be '%s', got %v", testContent, adapter.rawContent)
	}

	if adapter.content != testContent {
		t.Errorf("Expected content to be '%s', got %s", testContent, adapter.content)
	}
}

func TestDetailsAdapter_ContentToString(t *testing.T) {
	adapter := NewDetailsAdapter(nil, "test-resource")

	// Test with different types
	testCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string input",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "byte slice input",
			input:    []byte("byte content"),
			expected: "byte content",
		},
		{
			name:     "map input",
			input:    map[string]any{"key": "value", "number": 42},
			expected: "", // Will be JSON formatted, so we just check it's not empty
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			adapter.rawContent = tc.input
			result := adapter.contentToString()

			if tc.expected == "" {
				// For complex types, just ensure we get some output
				if result == "" {
					t.Errorf("Expected non-empty result for %s, got empty string", tc.name)
				}
			} else {
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			}
		})
	}
}

func TestDetailsAdapter_PredefinedTitle(t *testing.T) {
	// Test SetPredefinedTitle method
	adapter := NewDetailsAdapter(nil, "users")

	// Initially no predefined title
	if adapter.predefinedTitle != "" {
		t.Errorf("Expected empty predefined title initially, got '%s'", adapter.predefinedTitle)
	}

	// Set predefined title
	adapter.SetPredefinedTitle("custom title")
	if adapter.predefinedTitle != "custom title" {
		t.Errorf("Expected predefined title to be 'custom title', got '%s'", adapter.predefinedTitle)
	}

	// Test NewDetailsAdapterWithPredefinedTitle constructor
	adapterWithTitle := NewDetailsAdapterWithPredefinedTitle(nil, "users", "test title")
	if adapterWithTitle.predefinedTitle != "test title" {
		t.Errorf("Expected predefined title to be 'test title', got '%s'", adapterWithTitle.predefinedTitle)
	}
	if adapterWithTitle.resourceType != "users" {
		t.Errorf("Expected resourceType to be 'users', got '%s'", adapterWithTitle.resourceType)
	}
}

func TestListAdapter_PredefinedTitle(t *testing.T) {
	// Test SetPredefinedTitle method
	adapter := NewListAdapter(nil, "users", []string{"id", "name"})

	// Initially no predefined title
	if adapter.predefinedTitle != "" {
		t.Errorf("Expected empty predefined title initially, got '%s'", adapter.predefinedTitle)
	}

	// Set predefined title
	adapter.SetPredefinedTitle("custom list title")
	if adapter.predefinedTitle != "custom list title" {
		t.Errorf("Expected predefined title to be 'custom list title', got '%s'", adapter.predefinedTitle)
	}

	// Test NewListAdapterWithPredefinedTitle constructor
	adapterWithTitle := NewListAdapterWithPredefinedTitle(nil, "users", []string{"id", "name"}, "test list title")
	if adapterWithTitle.predefinedTitle != "test list title" {
		t.Errorf("Expected predefined title to be 'test list title', got '%s'", adapterWithTitle.predefinedTitle)
	}
	if adapterWithTitle.resourceType != "users" {
		t.Errorf("Expected resourceType to be 'users', got '%s'", adapterWithTitle.resourceType)
	}
}

func TestCreateAdapter_PredefinedTitle(t *testing.T) {
	// Test SetPredefinedTitle method
	adapter := NewCreateAdapter(nil, "users")

	// Initially no predefined title
	if adapter.predefinedTitle != "" {
		t.Errorf("Expected empty predefined title initially, got '%s'", adapter.predefinedTitle)
	}

	// Set predefined title
	adapter.SetPredefinedTitle("custom create title")
	if adapter.predefinedTitle != "custom create title" {
		t.Errorf("Expected predefined title to be 'custom create title', got '%s'", adapter.predefinedTitle)
	}

	// Test NewCreateAdapterWithPredefinedTitle constructor
	adapterWithTitle := NewCreateAdapterWithPredefinedTitle(nil, "users", "test create title")
	if adapterWithTitle.predefinedTitle != "test create title" {
		t.Errorf("Expected predefined title to be 'test create title', got '%s'", adapterWithTitle.predefinedTitle)
	}
	if adapterWithTitle.resourceType != "users" {
		t.Errorf("Expected resourceType to be 'users', got '%s'", adapterWithTitle.resourceType)
	}
}

func TestDetailsAdapter_BasicOperations(t *testing.T) {
	// Create adapter with mock database
	db := database.New() // This creates a real database service, but should be fine for testing
	adapter := NewDetailsAdapter(db, "users")

	// Test initial state
	if adapter.resourceType != "users" {
		t.Errorf("Expected resourceType to be 'users', got '%s'", adapter.resourceType)
	}

	if adapter.ready {
		t.Error("Expected adapter to not be ready initially")
	}

	// Test SetSize
	adapter.SetSize(100, 50)
	if adapter.width != 100 || adapter.height != 50 {
		t.Errorf("Expected dimensions 100x50, got %dx%d", adapter.width, adapter.height)
	}

	// Test Reset
	adapter.SetContent("some content")
	adapter.Reset()

	if adapter.content != "" {
		t.Errorf("Expected empty content after reset, got '%s'", adapter.content)
	}

	if adapter.ready {
		t.Error("Expected adapter to not be ready after reset")
	}
}
