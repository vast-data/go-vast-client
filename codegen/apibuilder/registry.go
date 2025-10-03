package apibuilder

import (
	"github.com/vast-data/go-vast-client/codegen/markers"
)

// RegisterAPITypedMarkers registers all apibuilder markers with the given registry
func RegisterAPITypedMarkers(registry *markers.Registry) error {
	// Register requestUrl markers for different HTTP methods
	// These use string type and we'll parse the method from the marker name
	if err := registry.Register("apityped:requestUrl:GET", markers.DescribesType, "",
		"Specifies GET request URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:requestUrl:POST", markers.DescribesType, "",
		"Specifies POST request URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:requestUrl:PUT", markers.DescribesType, "",
		"Specifies PUT request URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:requestUrl:DELETE", markers.DescribesType, "",
		"Specifies DELETE request URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:requestUrl:PATCH", markers.DescribesType, "",
		"Specifies PATCH request URL for this type"); err != nil {
		return err
	}

	// Register responseUrl markers for different HTTP methods
	if err := registry.Register("apityped:responseUrl:GET", markers.DescribesType, "",
		"Specifies GET response URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseUrl:POST", markers.DescribesType, "",
		"Specifies POST response URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseUrl:PUT", markers.DescribesType, "",
		"Specifies PUT response URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseUrl:DELETE", markers.DescribesType, "",
		"Specifies DELETE response URL for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseUrl:PATCH", markers.DescribesType, "",
		"Specifies PATCH response URL for this type"); err != nil {
		return err
	}

	// Register details markers (generates SearchParams + DetailsModel)
	if err := registry.Register("apityped:details:GET", markers.DescribesType, "",
		"Specifies GET details - generates SearchParams from query params and DetailsModel from GET response"); err != nil {
		return err
	}

	if err := registry.Register("apityped:details:PATCH", markers.DescribesType, "",
		"Specifies PATCH details - generates SearchParams from query params and DetailsModel from PATCH response"); err != nil {
		return err
	}

	// Register upsert markers (generates RequestBody + UpsertModel)
	if err := registry.Register("apityped:upsert:POST", markers.DescribesType, "",
		"Specifies POST upsert - generates RequestBody from POST request body and UpsertModel from POST response"); err != nil {
		return err
	}

	if err := registry.Register("apityped:upsert:PUT", markers.DescribesType, "",
		"Specifies PUT upsert - generates RequestBody from PUT request body and UpsertModel from PUT response"); err != nil {
		return err
	}

	if err := registry.Register("apityped:upsert:PATCH", markers.DescribesType, "",
		"Specifies PATCH upsert - generates RequestBody from PATCH request body and UpsertModel from PATCH response"); err != nil {
		return err
	}

	// Register responseBody markers
	if err := registry.Register("apityped:responseBody:GET", markers.DescribesType, "",
		"Specifies GET response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseBody:POST", markers.DescribesType, "",
		"Specifies POST response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseBody:PUT", markers.DescribesType, "",
		"Specifies PUT response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseBody:DELETE", markers.DescribesType, "",
		"Specifies DELETE response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseBody:PATCH", markers.DescribesType, "",
		"Specifies PATCH response body schema for this type"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseBody:SCHEMA", markers.DescribesType, "",
		"Specifies schema-based response body for this type"); err != nil {
		return err
	}

	// Register model markers
	if err := registry.Register("apityped:requestModel", markers.DescribesType, "",
		"Specifies the request model type for this endpoint"); err != nil {
		return err
	}

	if err := registry.Register("apityped:responseModel", markers.DescribesType, "",
		"Specifies the response model type for this endpoint"); err != nil {
		return err
	}

	// Register extraMethod markers for typed resources
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	for _, method := range methods {
		markerName := "apityped:extraMethod:" + method
		if err := registry.Register(markerName, markers.DescribesType, "",
			"Specifies an extra "+method+" method for typed resources"); err != nil {
			return err
		}
	}

	return nil
}

// MustRegisterAPITypedMarkers is like RegisterAPITypedMarkers but panics on error
func MustRegisterAPITypedMarkers(registry *markers.Registry) {
	if err := RegisterAPITypedMarkers(registry); err != nil {
		panic(err)
	}
}
