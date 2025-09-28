package apibuilder

import (
	"github.com/vast-data/go-vast-client/autogen/markers"
)

// RegisterAPIBuilderMarkers registers all apibuilder markers with the given registry
func RegisterAPIBuilderMarkers(registry *markers.Registry) error {
	// Register requestUrl markers for different HTTP methods
	// These use string type and we'll parse the method from the marker name
	if err := registry.Register("apibuilder:requestUrl:GET", markers.DescribesType, "",
		"Specifies GET request URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestUrl:POST", markers.DescribesType, "",
		"Specifies POST request URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestUrl:PUT", markers.DescribesType, "",
		"Specifies PUT request URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestUrl:DELETE", markers.DescribesType, "",
		"Specifies DELETE request URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestUrl:PATCH", markers.DescribesType, "",
		"Specifies PATCH request URL for this type"); err != nil {
		return err
	}

	// Register responseUrl markers for different HTTP methods
	if err := registry.Register("apibuilder:responseUrl:GET", markers.DescribesType, "",
		"Specifies GET response URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseUrl:POST", markers.DescribesType, "",
		"Specifies POST response URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseUrl:PUT", markers.DescribesType, "",
		"Specifies PUT response URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseUrl:DELETE", markers.DescribesType, "",
		"Specifies DELETE response URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseUrl:PATCH", markers.DescribesType, "",
		"Specifies PATCH response URL for this type"); err != nil {
		return err
	}

	// Register searchQuery markers
	if err := registry.Register("apibuilder:searchQuery:GET", markers.DescribesType, "",
		"Specifies GET search query parameters for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:searchQuery:SCHEMA", markers.DescribesType, "",
		"Specifies schema-based search query parameters for this type"); err != nil {
		return err
	}

	// Register requestBody markers
	if err := registry.Register("apibuilder:requestBody:POST", markers.DescribesType, "",
		"Specifies POST request body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestBody:PUT", markers.DescribesType, "",
		"Specifies PUT request body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestBody:PATCH", markers.DescribesType, "",
		"Specifies PATCH request body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestBody:SCHEMA", markers.DescribesType, "",
		"Specifies schema-based request body for this type"); err != nil {
		return err
	}

	// Register responseBody markers
	if err := registry.Register("apibuilder:responseBody:GET", markers.DescribesType, "",
		"Specifies GET response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseBody:POST", markers.DescribesType, "",
		"Specifies POST response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseBody:PUT", markers.DescribesType, "",
		"Specifies PUT response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseBody:DELETE", markers.DescribesType, "",
		"Specifies DELETE response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseBody:PATCH", markers.DescribesType, "",
		"Specifies PATCH response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseBody:SCHEMA", markers.DescribesType, "",
		"Specifies schema-based response body for this type"); err != nil {
		return err
	}

	// Register model markers
	if err := registry.Register("apibuilder:requestModel", markers.DescribesType, "",
		"Specifies the request model type for this endpoint"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:responseModel", markers.DescribesType, "",
		"Specifies the response model type for this endpoint"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:readOnly", markers.DescribesType, "",
		"Marks a resource as read-only (no create/update/delete operations)"); err != nil {
		return err
	}

	// New marker names (requestBody and model)
	if err := registry.Register("apibuilder:requestBody:POST", markers.DescribesType, "",
		"Specifies POST request body for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestBody:PUT", markers.DescribesType, "",
		"Specifies PUT request body for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestBody:PATCH", markers.DescribesType, "",
		"Specifies PATCH request body for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:requestBody:SCHEMA", markers.DescribesType, "",
		"Specifies request body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:model:GET", markers.DescribesType, "",
		"Specifies GET model for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:model:POST", markers.DescribesType, "",
		"Specifies POST model for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:model:PUT", markers.DescribesType, "",
		"Specifies PUT model for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:model:DELETE", markers.DescribesType, "",
		"Specifies DELETE model for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:model:PATCH", markers.DescribesType, "",
		"Specifies PATCH model for this type"); err != nil {
		return err
	}

	if err := registry.Register("apibuilder:model:SCHEMA", markers.DescribesType, "",
		"Specifies model schema for this type"); err != nil {
		return err
	}

	return nil
}

// MustRegisterAPIBuilderMarkers is like RegisterAPIBuilderMarkers but panics on error
func MustRegisterAPIBuilderMarkers(registry *markers.Registry) {
	if err := RegisterAPIBuilderMarkers(registry); err != nil {
		panic(err)
	}
}
