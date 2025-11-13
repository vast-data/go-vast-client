package core

// ExtraMethodMetadata stores the URL path and HTTP verb for extra methods
// This should be generated at compile time by the code generator
type ExtraMethodMetadata struct {
	MethodName string // e.g., "ApiTokenRevoke_PATCH"
	HTTPVerb   string // e.g., "PATCH"
	URLPath    string // e.g., "/apitokens/{id}/revoke/"
	Summary    string // e.g., "Revoke API Token"
}

// ExtraMethodRegistry is a global registry of extra method metadata
// This is populated by code generation during build time
var ExtraMethodRegistry = map[string]map[string]ExtraMethodMetadata{
	// Key is resource type (e.g., "apitokens")
	// Value is map of method name to metadata
}

// RegisterExtraMethod registers metadata for an extra method
// This is called by generated init() functions
func RegisterExtraMethod(resourceType, methodName, httpVerb, urlPath, summary string) {
	if ExtraMethodRegistry[resourceType] == nil {
		ExtraMethodRegistry[resourceType] = make(map[string]ExtraMethodMetadata)
	}
	ExtraMethodRegistry[resourceType][methodName] = ExtraMethodMetadata{
		MethodName: methodName,
		HTTPVerb:   httpVerb,
		URLPath:    urlPath,
		Summary:    summary,
	}
}

// GetExtraMethodMetadata retrieves metadata for a specific extra method
func GetExtraMethodMetadata(resourceType, methodName string) (ExtraMethodMetadata, bool) {
	if methods, ok := ExtraMethodRegistry[resourceType]; ok {
		metadata, found := methods[methodName]
		return metadata, found
	}
	return ExtraMethodMetadata{}, false
}

// GetAllExtraMethodsForResource returns all extra methods for a resource type
func GetAllExtraMethodsForResource(resourceType string) []ExtraMethodMetadata {
	if methods, ok := ExtraMethodRegistry[resourceType]; ok {
		result := make([]ExtraMethodMetadata, 0, len(methods))
		for _, metadata := range methods {
			result = append(result, metadata)
		}
		return result
	}
	return nil
}
