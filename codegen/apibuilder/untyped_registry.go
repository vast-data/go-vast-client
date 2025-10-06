package apibuilder

import (
	"fmt"

	"github.com/vast-data/go-vast-client/codegen/markers"
)

// RegisterAPIUntypedMarkers registers all apiuntyped markers with the given registry
func RegisterAPIUntypedMarkers(registry *markers.Registry) error {
	// Register extraMethod markers for each HTTP method
	// The value is expected as a string path
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	for _, method := range methods {
		// Register apiuntyped:extraMethod:METHOD markers
		markerName := "apiuntyped:extraMethod:" + method
		if err := registry.Register(markerName, markers.DescribesType, "",
			"Specifies an extra "+method+" method for untyped resources"); err != nil {
			return err
		}

		// Register apiall:extraMethod:METHOD markers (for both typed and untyped)
		markerName = "apiall:extraMethod:" + method
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
			markerName := fmt.Sprintf("apiuntyped:extraMethod[wait(%s)]:%s", timeout, method)
			if err := registry.Register(markerName, markers.DescribesType, "",
				fmt.Sprintf("Specifies an async %s method with %s wait timeout for untyped resources", method, timeout)); err != nil {
				return err
			}
			markerName = fmt.Sprintf("apiall:extraMethod[wait(%s)]:%s", timeout, method)
			if err := registry.Register(markerName, markers.DescribesType, "",
				fmt.Sprintf("Specifies an async %s method with %s wait timeout for untyped and typed resources", method, timeout)); err != nil {
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
		if err := registry.Register("apiuntyped:extraMethod:"+combo, markers.DescribesType, "",
			"Specifies multiple extra methods ("+combo+") for untyped resources"); err != nil {
			return err
		}
		if err := registry.Register("apiall:extraMethod:"+combo, markers.DescribesType, "",
			"Specifies multiple extra methods ("+combo+") for both typed and untyped resources"); err != nil {
			return err
		}
	}

	return nil
}

// MustRegisterAPIUntypedMarkers is like RegisterAPIUntypedMarkers but panics on error
func MustRegisterAPIUntypedMarkers(registry *markers.Registry) {
	if err := RegisterAPIUntypedMarkers(registry); err != nil {
		panic(err)
	}
}
