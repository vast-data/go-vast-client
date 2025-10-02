package markers

import (
	"go/ast"
	"go/token"
	"reflect"
)

// TargetType describes which kind of Go construct a marker can be applied to.
type TargetType int

const (
	// DescribesPackage indicates that a marker is associated with a package.
	DescribesPackage TargetType = iota
	// DescribesType indicates that a marker is associated with a type declaration.
	DescribesType
	// DescribesField indicates that a marker is associated with a struct field.
	DescribesField
)

func (t TargetType) String() string {
	switch t {
	case DescribesPackage:
		return "package"
	case DescribesType:
		return "type"
	case DescribesField:
		return "field"
	default:
		return "unknown"
	}
}

// ArgumentType represents the type of marker arguments.
type ArgumentType int

const (
	// InvalidType represents a type that can't be parsed.
	InvalidType ArgumentType = iota
	// StringType is a string argument.
	StringType
	// IntType is an integer argument.
	IntType
	// BoolType is a boolean argument.
	BoolType
	// SliceType is a slice argument.
	SliceType
	// MapType is a map argument with string keys.
	MapType
	// AnyType matches any type (interface{}).
	AnyType
)

// Argument describes the type and properties of a marker argument.
type Argument struct {
	// Type is the type of this argument.
	Type ArgumentType
	// Optional indicates if this argument is optional.
	Optional bool
	// ItemType is the type of slice items or map values.
	ItemType *Argument
}

// Definition defines how to parse a specific marker.
type Definition struct {
	// Name is the marker's name (e.g., "vast:generate").
	Name string
	// Target indicates which Go constructs this marker can be applied to.
	Target TargetType
	// OutputType is the Go type that this marker parses into.
	OutputType reflect.Type
	// Fields maps argument names to their types (for struct outputs).
	Fields map[string]Argument
	// FieldNames maps argument names to struct field names.
	FieldNames map[string]string
	// Description provides help text for this marker.
	Description string
}

// MarkerValue represents a parsed marker with its associated AST node.
type MarkerValue struct {
	// Name is the marker name.
	Name string `json:"name"`
	// Value is the parsed marker value.
	Value interface{} `json:"value"`
	// Node is the AST node this marker is associated with.
	Node ast.Node `json:"-"`
	// Target indicates what type of construct this marker describes.
	Target TargetType `json:"target"`
	// Position is the source position of the marker comment.
	Position token.Pos `json:"position"`
}

// MarkerValues maps marker names to their parsed values.
type MarkerValues map[string][]interface{}

// Get returns the first value for the given marker name, or nil if not found.
func (v MarkerValues) Get(name string) interface{} {
	vals := v[name]
	if len(vals) == 0 {
		return nil
	}
	return vals[0]
}

// GetAll returns all values for the given marker name.
func (v MarkerValues) GetAll(name string) []interface{} {
	return v[name]
}

// Has returns true if the marker name exists (even with empty values).
func (v MarkerValues) Has(name string) bool {
	_, exists := v[name]
	return exists
}

// TypeInfo contains information about a parsed Go type and its markers.
type TypeInfo struct {
	// Name is the type name.
	Name string
	// Markers are the markers associated with this type.
	Markers MarkerValues
	// Fields are the struct fields (if this is a struct type).
	Fields []FieldInfo
	// Doc is the documentation comment.
	Doc string
	// RawDecl is the raw AST declaration.
	RawDecl *ast.GenDecl
	// RawSpec is the raw AST type spec.
	RawSpec *ast.TypeSpec
	// RawFile is the raw AST file.
	RawFile *ast.File
}

// FieldInfo contains information about a struct field and its markers.
type FieldInfo struct {
	// Name is the field name.
	Name string
	// Markers are the markers associated with this field.
	Markers MarkerValues
	// Tag is the struct tag.
	Tag reflect.StructTag
	// Doc is the documentation comment.
	Doc string
	// RawField is the raw AST field.
	RawField *ast.Field
}

// TypeCallback is called for each type found during parsing.
type TypeCallback func(*TypeInfo)

// markerComment represents a comment that contains a marker.
type markerComment struct {
	*ast.Comment
	fromGodoc bool
}

// Text returns the marker text, stripped of comment prefix and whitespace.
func (c markerComment) Text() string {
	text := c.Comment.Text
	if len(text) < 2 {
		return ""
	}
	// Remove "//" prefix and trim whitespace
	return text[2:]
}
