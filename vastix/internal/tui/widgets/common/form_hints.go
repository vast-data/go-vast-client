package common

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"vastix/internal/logging"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/vast-data/go-vast-client/openapi_schema"
)

type InputDefinitionFrom int

var (
	InputDefinitionFromCreate      InputDefinitionFrom = 1
	InputDefinitionFromRead        InputDefinitionFrom = 2
	InputDefinitionFromQueryParams InputDefinitionFrom = 3
)

// FormHints defines metadata and overrides for TUI form generation.
// Contains schema reference for OpenAPI-based input generation and custom inputs
// that should be appended to the schema-derived inputs.
type FormHints struct {
	// SchemaRef defines where to get request (create) and response (read) schemas from OpenAPI
	SchemaRef *SchemaReference

	// CustomInputs defines additional input fields that should be appended to
	// the inputs generated from the OpenAPI schema. These inputs will be added
	// after all schema-derived inputs.
	CustomInputs []InputDefinition
}

// OpenAPIEndpointRef defines a reference to a specific HTTP method + path
// in the OpenAPI schema, used for schema extraction.
type OpenAPIEndpointRef struct {
	// HTTP method (e.g., "get", "post", "patch")
	Method string
	// Path in OpenAPI (e.g., "/volumes", "/volumes/{id}")
	Path string
}

// Removed FormHintsForCustom as it was unused in TUI

type InputDefinition struct {
	Name        string
	Type        string
	Required    bool
	Description string
	Default     interface{}
	Enum        []string
	Format      string
	Minimum     *float64
	Maximum     *float64
	Placeholder string // Custom placeholder text for input fields
	// New fields for nested objects
	Properties map[string]*InputDefinition
	Items      *InputDefinition // For arrays
}

// SchemaReference encapsulates both create and read endpoints for a resource.
// Used to extract the POST request schema (for resources) and the GET response schema (for resources or data sources).
type SchemaReference struct {
	// Create specifies the OpenAPI endpoint to use for extracting the creation schema (e.g., POST /volumes).
	Create *OpenAPIEndpointRef
	Read   *OpenAPIEndpointRef
}

func NewSchemaReference(
	createMethod, createPath string,
	readMethod, readPath string,
) *SchemaReference {
	var createRef *OpenAPIEndpointRef

	createRef = &OpenAPIEndpointRef{
		Method: createMethod,
		Path:   createPath,
	}

	readRef := &OpenAPIEndpointRef{
		Method: readMethod,
		Path:   readPath,
	}

	return &SchemaReference{
		Create: createRef,
		Read:   readRef,
	}
}

func (sr *SchemaReference) GetCreatePath() string {
	return sr.Create.Path
}

func (sr *SchemaReference) GetReadPath() string {
	return sr.Read.Path
}

func (sr *SchemaReference) getSchemaFromReadRef() (*openapi3.SchemaRef, error) {
	resourcePath := sr.GetReadPath()
	method := sr.Read.Method
	var schemaRef *openapi3.SchemaRef
	var err error

	if resourcePath == "" {
		return nil, fmt.Errorf("resource path is empty for read schema")
	}
	if method == "" {
		return nil, fmt.Errorf("HTTP method is empty for read schema")
	}

	switch method {
	case http.MethodGet:
		if schemaRef, err = openapi_schema.GetResponseModelSchema(http.MethodGet, resourcePath); err != nil {
			return nil, fmt.Errorf("failed to get schema for GET %q: %w", resourcePath, err)
		}
	default:
		return nil, fmt.Errorf(
			"not supported resource method %q for schema generation", method,
		)
	}
	return schemaRef, nil
}

func (sr *SchemaReference) getSchemaFromCreateRef() (*openapi3.SchemaRef, error) {
	resourcePath := sr.GetCreatePath()
	method := sr.Create.Method
	var schemaRef *openapi3.SchemaRef
	var err error

	if resourcePath == "" {
		return nil, fmt.Errorf("resource path is empty for create schema")
	}
	if method == "" {
		return nil, fmt.Errorf("HTTP method is empty for create schema")
	}

	switch method {
	case http.MethodPost:
		if schemaRef, err = openapi_schema.GetRequestBodySchema(http.MethodPost, resourcePath); err != nil {
			return nil, fmt.Errorf("failed to get POST schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodGet:
		if schemaRef, err = openapi_schema.GetResponseModelSchema(http.MethodGet, resourcePath); err != nil {
			return nil, fmt.Errorf("failed to get GET schema for resource %q: %w", resourcePath, err)
		}
	case http.MethodPatch:
		if schemaRef, err = openapi_schema.GetRequestBodySchema(http.MethodPatch, resourcePath); err != nil {
			return nil, fmt.Errorf("failed to get PATCH schema for resource %q: %w", resourcePath, err)
		}
	default:
		return nil, fmt.Errorf(
			"unsupported method %q for resource %q (CreateSchema)", method, resourcePath,
		)
	}
	return schemaRef, nil
}

func (sr *SchemaReference) getSchemaFromQueryParamsRef() (*openapi3.SchemaRef, error) {
	resourcePath := sr.GetReadPath()
	var schemaRef *openapi3.SchemaRef
	var err error

	if resourcePath == "" {
		return nil, fmt.Errorf("resource path is empty for query params schema")
	}

	if schemaRef, err = openapi_schema.GetSchema_GET_QueryParams(resourcePath); err != nil {
		return nil, fmt.Errorf("failed to get query params schema for %q: %w", resourcePath, err)
	}
	return schemaRef, nil
}

// getInputDefinitionsFromSchema extracts input definitions from an OpenAPI schema for form generation
func (sr *SchemaReference) getInputDefinitionsFromSchema(from InputDefinitionFrom) ([]InputDefinition, error) {
	var (
		schema *openapi3.SchemaRef
		err    error
	)

	switch from {
	case InputDefinitionFromCreate:
		schema, err = sr.getSchemaFromCreateRef()
	case InputDefinitionFromRead:
		schema, err = sr.getSchemaFromReadRef()
	case InputDefinitionFromQueryParams:
		schema, err = sr.getSchemaFromQueryParamsRef()
	default:
		panic(fmt.Sprintf("unknown input definition from (%q)", from))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	if schema == nil || schema.Value == nil {
		return []InputDefinition{}, nil
	}

	resolvedSchema := openapi_schema.ResolveComposedSchema(schema.Value)
	if resolvedSchema == nil || resolvedSchema.Properties == nil {
		return []InputDefinition{}, nil
	}

	var inputDefs []InputDefinition
	requiredFields := make(map[string]bool)

	// Build set of required fields
	for _, req := range resolvedSchema.Required {
		requiredFields[req] = true
	}

	// Process each property
	for propName, propSchemaRef := range resolvedSchema.Properties {
		if propSchemaRef == nil || propSchemaRef.Value == nil {
			continue
		}

		propSchema := openapi_schema.ResolveComposedSchema(openapi_schema.ResolveAllRefs(propSchemaRef))
		if propSchema == nil {
			continue
		}

		// Skip read-only fields
		if propSchema.ReadOnly {
			continue
		}

		// Skip ambiguous objects (objects with no properties)
		if isAmbiguousObject(propSchema) {
			continue
		}

		inputDef := convertSchemaToInputDefinition(propName, propSchema, requiredFields[propName])
		inputDefs = append(inputDefs, inputDef)
	}

	return inputDefs, nil
}

// GetInputsFromCreateSchema generates InputWrapper instances from OpenAPI create schema definitions
func (sr *SchemaReference) GetInputsFromCreateSchema(onlyPrimitives bool) (Inputs, error) {
	inputDefs, err := sr.getInputDefinitionsFromSchema(InputDefinitionFromCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to get input definitions: %w", err)
	}
	return inputDefsToInputs(inputDefs, onlyPrimitives)
}

// GetInputsFromCreateSchemaWithCustom generates InputWrapper instances from OpenAPI create schema
// and appends any custom inputs defined in FormHints
func (fh *FormHints) GetInputsFromCreateSchemaWithCustom(onlyPrimitives bool) (Inputs, error) {
	// Get inputs from schema if available
	var schemaInputDefs []InputDefinition
	if fh.SchemaRef != nil {
		var err error
		schemaInputDefs, err = fh.SchemaRef.getInputDefinitionsFromSchema(InputDefinitionFromCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema input definitions: %w", err)
		}
	}

	// Append custom inputs
	allInputDefs := append(schemaInputDefs, fh.CustomInputs...)

	return inputDefsToInputs(allInputDefs, onlyPrimitives)
}

// GetInputsFromReadSchema generates InputWrapper instances from OpenAPI read schema definitions
func (sr *SchemaReference) GetInputsFromReadSchema(onlyPrimitives bool) (Inputs, error) {
	inputDefs, err := sr.getInputDefinitionsFromSchema(InputDefinitionFromRead)
	if err != nil {
		return nil, fmt.Errorf("failed to get input definitions: %w", err)
	}
	return inputDefsToInputs(inputDefs, onlyPrimitives)
}

// GetInputsFromQueryParams generates InputWrapper instances from OpenAPI query parameters
func (sr *SchemaReference) GetInputsFromQueryParams(onlyPrimitives bool) (Inputs, error) {
	inputDefs, err := sr.getInputDefinitionsFromSchema(InputDefinitionFromQueryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get input definitions: %w", err)
	}
	return inputDefsToInputs(inputDefs, onlyPrimitives)
}

// GetInputsFromQueryParamsWithCustom generates InputWrapper instances from OpenAPI query parameters
// and appends any custom inputs defined in FormHints
func (fh *FormHints) GetInputsFromQueryParamsWithCustom(onlyPrimitives bool) (Inputs, error) {
	// Get inputs from schema if available
	var schemaInputDefs []InputDefinition
	if fh.SchemaRef != nil {
		var err error
		schemaInputDefs, err = fh.SchemaRef.getInputDefinitionsFromSchema(InputDefinitionFromQueryParams)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema input definitions: %w", err)
		}
	}

	// Append custom inputs
	allInputDefs := append(schemaInputDefs, fh.CustomInputs...)

	return inputDefsToInputs(allInputDefs, onlyPrimitives)
}

// isAmbiguousObject checks if an object has no properties defined (should be skipped)
func isAmbiguousObject(schema *openapi3.Schema) bool {
	if schema == nil || schema.Type == nil || len(*schema.Type) == 0 {
		return false
	}

	// Check if it's an object type with no properties and no additionalProperties (map types)
	return (*schema.Type)[0] == openapi3.TypeObject && len(schema.Properties) == 0 && schema.AdditionalProperties.Schema == nil
}

// convertSchemaToInputDefinition recursively converts an OpenAPI schema to an InputDefinition
func convertSchemaToInputDefinition(name string, schema *openapi3.Schema, required bool) InputDefinition {
	inputDef := InputDefinition{
		Name:        name,
		Required:    required,
		Description: schema.Description,
		Default:     schema.Default,
		Format:      schema.Format,
	}

	// Handle enum values
	if len(schema.Enum) > 0 {
		for _, enumVal := range schema.Enum {
			if strVal, ok := enumVal.(string); ok {
				inputDef.Enum = append(inputDef.Enum, strVal)
			}
		}
	}

	// Handle numeric constraints
	if schema.Min != nil {
		inputDef.Minimum = schema.Min
	}
	if schema.Max != nil {
		inputDef.Maximum = schema.Max
	}

	// Determine input type based on OpenAPI type
	if schema.Type != nil && len(*schema.Type) > 0 {
		switch (*schema.Type)[0] {
		case openapi3.TypeString:
			inputDef.Type = "string"
		case openapi3.TypeInteger:
			inputDef.Type = "integer"
		case openapi3.TypeNumber:
			inputDef.Type = "number"
		case openapi3.TypeBoolean:
			inputDef.Type = "boolean"
		case openapi3.TypeArray:
			inputDef.Type = "array"
			// Handle array items recursively
			if schema.Items != nil && schema.Items.Value != nil {
				itemSchema := openapi_schema.ResolveComposedSchema(openapi_schema.ResolveAllRefs(schema.Items))
				if itemSchema != nil {
					itemDef := convertSchemaToInputDefinition("item", itemSchema, false)
					inputDef.Items = &itemDef
				}
			}
		case openapi3.TypeObject:
			inputDef.Type = "object"
			// Handle object properties recursively
			if schema.Properties != nil {
				inputDef.Properties = make(map[string]*InputDefinition)

				// Build required fields set for this object
				objectRequired := make(map[string]bool)
				for _, req := range schema.Required {
					objectRequired[req] = true
				}

				// Process each property recursively
				for propName, propSchemaRef := range schema.Properties {
					if propSchemaRef == nil || propSchemaRef.Value == nil {
						continue
					}

					propSchema := openapi_schema.ResolveComposedSchema(openapi_schema.ResolveAllRefs(propSchemaRef))
					if propSchema == nil || propSchema.ReadOnly {
						continue
					}

					// Skip ambiguous objects (objects with no properties)
					if isAmbiguousObject(propSchema) {
						continue
					}

					propDef := convertSchemaToInputDefinition(propName, propSchema, objectRequired[propName])
					inputDef.Properties[propName] = &propDef
				}
			}
		default:
			inputDef.Type = "string" // Default fallback
		}
	} else {
		inputDef.Type = "string" // Default fallback
	}

	return inputDef
}

func inputDefsToInputs(defs []InputDefinition, onlyPrimitives bool) (Inputs, error) {
	var inputs Inputs
	auxlog := logging.GetAuxLogger()

	for _, def := range defs {
		input, err := createInputFromDefinition(def, onlyPrimitives)
		if err != nil {
			var skipErr SkipDefinition
			if errors.As(err, &skipErr) {
				auxlog.Printf("<<skipping definition>> '%s': %s", def.Name, skipErr.reason)
				continue // Skip this definition
			}

			return nil, fmt.Errorf("failed to create input from definition %s: %w", def.Name, err)
		}
		inputs = append(inputs, input)
	}

	sort.Sort(inputs)
	return inputs, nil
}
