package markers

import (
	"os"
	"reflect"
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	// Test registering a simple marker
	err := registry.Register("test:simple", DescribesType, struct{}{}, "Simple test marker")
	if err != nil {
		t.Fatalf("Failed to register simple marker: %v", err)
	}

	// Test registering a complex marker
	type ComplexMarker struct {
		Name    string `marker:"name"`
		Count   int    `marker:"count,optional"`
		Enabled bool   `marker:"enabled,optional"`
	}

	err = registry.Register("test:complex", DescribesField, ComplexMarker{}, "Complex test marker")
	if err != nil {
		t.Fatalf("Failed to register complex marker: %v", err)
	}

	// Test lookup
	def := registry.Lookup("+test:simple", DescribesType)
	if def == nil {
		t.Fatal("Failed to lookup registered marker")
	}

	if def.Name != "test:simple" {
		t.Errorf("Expected marker name 'test:simple', got '%s'", def.Name)
	}
}

func TestDefinition_Parse(t *testing.T) {
	registry := NewRegistry()

	// Register test markers
	type TestMarker struct {
		Name    string `marker:"name"`
		Count   int    `marker:"count,optional"`
		Enabled bool   `marker:"enabled,optional"`
	}

	registry.MustRegister("test:marker", DescribesType, TestMarker{}, "Test marker")

	def := registry.GetDefinition("test:marker")
	if def == nil {
		t.Fatal("Test marker not found")
	}

	tests := []struct {
		name     string
		input    string
		expected TestMarker
		wantErr  bool
	}{
		{
			name:     "simple name",
			input:    "+test:marker=name=example",
			expected: TestMarker{Name: "example"},
		},
		{
			name:     "multiple args",
			input:    "+test:marker=name=example,count=5,enabled=true",
			expected: TestMarker{Name: "example", Count: 5, Enabled: true},
		},
		{
			name:     "quoted string",
			input:    `+test:marker=name="quoted example"`,
			expected: TestMarker{Name: "quoted example"},
		},
		{
			name:     "empty marker",
			input:    "+test:marker",
			expected: TestMarker{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := def.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("Parse() = %+v, expected %+v", result, tt.expected)
				}
			}
		})
	}
}

func TestCollector_ParseSource(t *testing.T) {
	registry := NewRegistry()
	registry.MustRegister("test:generate", DescribesType, struct{}{}, "Test generate marker")
	registry.MustRegister("test:required", DescribesField, struct{}{}, "Test required marker")

	collector := NewCollector(registry)

	source := `package test

// +test:generate
type User struct {
	// +test:required
	Name string ` + "`json:\"name\"`" + `
}
`

	markers, err := collector.ParseSource("test.go", source)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	if len(markers) != 2 {
		t.Errorf("Expected 2 markers, got %d", len(markers))
	}

	// Check that we found the expected markers
	foundGenerate := false
	foundRequired := false

	for _, marker := range markers {
		switch marker.Name {
		case "test:generate":
			foundGenerate = true
			if marker.Target != DescribesType {
				t.Errorf("Expected DescribesType for generate marker, got %v", marker.Target)
			}
		case "test:required":
			foundRequired = true
			if marker.Target != DescribesField {
				t.Errorf("Expected DescribesField for required marker, got %v", marker.Target)
			}
		}
	}

	if !foundGenerate {
		t.Error("Did not find test:generate marker")
	}
	if !foundRequired {
		t.Error("Did not find test:required marker")
	}
}

func TestCollector_EachType(t *testing.T) {
	registry := NewRegistry()
	registry.MustRegister("test:generate", DescribesType, struct{}{}, "Test generate marker")
	registry.MustRegister("test:required", DescribesField, struct{}{}, "Test required marker")

	collector := NewCollector(registry)

	source := `package test

// +test:generate
type User struct {
	// +test:required
	Name string ` + "`json:\"name\"`" + `
	
	Email string ` + "`json:\"email\"`" + `
}

type Product struct {
	ID string
}
`

	// Write source to temporary file for testing
	tmpFile := "test_temp.go"
	err := writeSourceToFile(tmpFile, source)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	defer removeFile(tmpFile)

	var foundTypes []string
	err = collector.EachType(tmpFile, func(typeInfo *TypeInfo) {
		foundTypes = append(foundTypes, typeInfo.Name)

		if typeInfo.Name == "User" {
			// Check that User has the generate marker
			if !typeInfo.Markers.Has("test:generate") {
				t.Error("User type should have test:generate marker")
			}

			// Check that User has 2 fields
			if len(typeInfo.Fields) != 2 {
				t.Errorf("Expected 2 fields for User, got %d", len(typeInfo.Fields))
			}

			// Check that Name field has required marker
			for _, field := range typeInfo.Fields {
				if field.Name == "Name" && !field.Markers.Has("test:required") {
					t.Error("Name field should have test:required marker")
				}
			}
		}
	})

	if err != nil {
		t.Fatalf("EachType failed: %v", err)
	}

	expectedTypes := []string{"User", "Product"}
	if !reflect.DeepEqual(foundTypes, expectedTypes) {
		t.Errorf("Expected types %v, got %v", expectedTypes, foundTypes)
	}
}

func TestArgumentTypes(t *testing.T) {
	registry := NewRegistry()

	type SliceMarker struct {
		Items []string `marker:"items"`
	}

	type MapMarker struct {
		Config map[string]string `marker:"config"`
	}

	registry.MustRegister("test:slice", DescribesType, SliceMarker{}, "Test slice marker")
	registry.MustRegister("test:map", DescribesType, MapMarker{}, "Test map marker")

	tests := []struct {
		name     string
		marker   string
		input    string
		expected interface{}
	}{
		{
			name:     "slice with braces",
			marker:   "test:slice",
			input:    "+test:slice=items={item1,item2,item3}",
			expected: SliceMarker{Items: []string{"item1", "item2", "item3"}},
		},
		{
			name:     "slice with semicolons",
			marker:   "test:slice",
			input:    "+test:slice=items=item1;item2;item3",
			expected: SliceMarker{Items: []string{"item1", "item2", "item3"}},
		},
		{
			name:     "map",
			marker:   "test:map",
			input:    "+test:map=config={key1:value1,key2:value2}",
			expected: MapMarker{Config: map[string]string{"key1": "value1", "key2": "value2"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := registry.GetDefinition(tt.marker)
			if def == nil {
				t.Fatalf("Marker %s not found", tt.marker)
			}

			result, err := def.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

// Helper functions for testing

func writeSourceToFile(filename, source string) error {
	return os.WriteFile(filename, []byte(source), 0644)
}

func removeFile(filename string) {
	os.Remove(filename)
}

func TestMarkerValues_Methods(t *testing.T) {
	values := make(MarkerValues)
	values["test"] = []interface{}{"value1", "value2"}
	values["empty"] = []interface{}{}

	// Test Get
	if got := values.Get("test"); got != "value1" {
		t.Errorf("Get() = %v, expected 'value1'", got)
	}

	if got := values.Get("empty"); got != nil {
		t.Errorf("Get() for empty slice = %v, expected nil", got)
	}

	if got := values.Get("nonexistent"); got != nil {
		t.Errorf("Get() for nonexistent = %v, expected nil", got)
	}

	// Test GetAll
	if got := values.GetAll("test"); !reflect.DeepEqual(got, []interface{}{"value1", "value2"}) {
		t.Errorf("GetAll() = %v, expected ['value1', 'value2']", got)
	}

	// Test Has
	if !values.Has("test") {
		t.Error("Has() for existing key should return true")
	}

	if !values.Has("empty") {
		t.Error("Has() for empty slice should return true")
	}

	if values.Has("nonexistent") {
		t.Error("Has() for nonexistent key should return false")
	}
}
