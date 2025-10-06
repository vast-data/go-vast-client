/*
Package markers provides utilities for parsing and processing "marker comments"
from Go source code, similar to controller-tools but tailored for go-vast-client.

Marker comments are special comments that start with `// +` and provide
metadata for code generation and processing.

# Basic Usage

	registry := markers.NewRegistry()

	// Register custom markers
	registry.Register("vast:generate", markers.DescribesType, GenerateConfig{})
	registry.Register("vast:field:required", markers.DescribesField, struct{}{})

	// Parse a Go file
	collector := &markers.Collector{Registry: registry}
	markers, err := collector.ParseFile("example.go")

# Marker Syntax

Markers follow the general form:

	// +prefix:module:submodule=value
	// +prefix:module:key=value,key2=value2
	// +prefix:module

Examples:

	// +vast:generate
	// +vast:field:required
	// +vast:validation:maxLength=50
	// +vast:api:endpoint="/users",method="POST"

# Supported Argument Types

- Strings: `name="value"` or `name=value`
- Integers: `count=42`
- Booleans: `enabled=true`
- Slices: `items={val1,val2,val3}` or `items=val1;val2;val3`
- Maps: `config={key1:value1,key2:value2}`

# Target Types

Markers can target different Go constructs:

- DescribesPackage: Package-level markers
- DescribesType: Type/struct declarations
- DescribesField: Struct field declarations
*/
package markers
