package apibuilder

import (
	"github.com/vast-data/go-vast-client/codegen/markers"
)

// ExtraMethod represents an extra method configuration for untyped resources
type ExtraMethod struct {
	Method string `json:"method"` // HTTP method (GET, POST, etc.)
	Path   string `json:"path"`   // API path (e.g., "/users/{id}/access_keys/")
}

// RegisterAPIUntypedMarkers registers all apiuntyped markers with the given registry
func RegisterAPIUntypedMarkers(registry *markers.Registry) error {
	// Register extraMethod markers for each HTTP method
	// The value is expected as a string path
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	
	for _, method := range methods {
		markerName := "apiuntyped:extraMethod:" + method
		if err := registry.Register(markerName, markers.DescribesType, "",
			"Specifies an extra "+method+" method for untyped resources"); err != nil {
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
