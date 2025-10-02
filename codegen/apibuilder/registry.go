package apibuilder

import (
	"fmt"

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

	// Register unified ops marker (replaces details + upsert)
	// Supports any combination of CRUD operations
	crudCombos := []string{
		"CRUD", "CRU", "CRD", "CUD", "CR", "CU", "CD", "RU", "RUD", "RD", "UD", "C", "R", "U", "D",
	}
	for _, combo := range crudCombos {
		markerName := "apityped:ops:" + combo
		if err := registry.Register(markerName, markers.DescribesType, "",
			"Specifies "+combo+" operations for this resource"); err != nil {
			return err
		}
	}

	// Keep legacy markers for backward compatibility (will be deprecated)
	// Register details markers (generates SearchParams + DetailsModel)
	if err := registry.Register("apityped:details:GET", markers.DescribesType, "",
		"[DEPRECATED] Specifies GET details - use apityped:ops:R instead"); err != nil {
		return err
	}

	if err := registry.Register("apityped:details:PATCH", markers.DescribesType, "",
		"[DEPRECATED] Specifies PATCH details - use apityped:ops:R instead"); err != nil {
		return err
	}

	// Register upsert markers (generates RequestBody + UpsertModel)
	if err := registry.Register("apityped:upsert:POST", markers.DescribesType, "",
		"[DEPRECATED] Specifies POST upsert - use apityped:ops:CU instead"); err != nil {
		return err
	}

	if err := registry.Register("apityped:upsert:PUT", markers.DescribesType, "",
		"[DEPRECATED] Specifies PUT upsert - use apityped:ops:CU instead"); err != nil {
		return err
	}

	if err := registry.Register("apityped:upsert:PATCH", markers.DescribesType, "",
		"[DEPRECATED] Specifies PATCH upsert - use apityped:ops:U instead"); err != nil {
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

	// Register apiall:extraMethod markers (generates both typed and untyped)
	for _, method := range methods {
		markerName := "apiall:extraMethod:" + method
		if err := registry.Register(markerName, markers.DescribesType, "",
			"Specifies an extra "+method+" method for both typed and untyped resources"); err != nil {
			return err
		}
	}

	// Register async task markers with common wait timeouts
	// Note: These are for marker collection. The actual timeout value is parsed from the marker text.
	commonTimeouts := []string{"1s", "5s", "10s", "30s", "1m", "3m", "5m", "10m", "15m", "30m", "1h", "2h", "3h", "6h", "12h", "24h", "1d"}
	for _, method := range methods {
		for _, timeout := range commonTimeouts {
			markerName := fmt.Sprintf("apityped:extraMethod[wait(%s)]:%s", timeout, method)
			if err := registry.Register(markerName, markers.DescribesType, "",
				fmt.Sprintf("Specifies an async %s method with %s wait timeout for typed resources", method, timeout)); err != nil {
				return err
			}
			markerName = fmt.Sprintf("apiall:extraMethod[wait(%s)]:%s", timeout, method)
			if err := registry.Register(markerName, markers.DescribesType, "",
				fmt.Sprintf("Specifies an async %s method with %s wait timeout for typed and untyped resources", method, timeout)); err != nil {
				return err
			}
		}
	}

	// Register common multi-method combinations (e.g., "POST|PATCH|DELETE")
	// These are shortcuts for declaring multiple methods at once
	commonCombinations := []string{
		"POST|PATCH|DELETE",
		"POST|PUT",
		"POST|PATCH",
		"GET|POST",
		"GET|PATCH",
		"PATCH|GET",
		"PATCH|DELETE",
		"PUT|DELETE",
	}
	for _, combo := range commonCombinations {
		if err := registry.Register("apityped:extraMethod:"+combo, markers.DescribesType, "",
			"Specifies multiple extra methods ("+combo+") for typed resources"); err != nil {
			return err
		}
		if err := registry.Register("apiall:extraMethod:"+combo, markers.DescribesType, "",
			"Specifies multiple extra methods ("+combo+") for both typed and untyped resources"); err != nil {
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
