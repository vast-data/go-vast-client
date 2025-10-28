// Package vast_client provides a Go client library for the VAST Data Management System (VMS) REST API.
//
// Two client types are available:
//   - VMSRest (untyped): Flexible map-based client. Recommended for most use cases.
//   - TypedVMSRest: Strongly-typed client with compile-time safety.
package vast_client

import (
	"github.com/vast-data/go-vast-client/core"
	"github.com/vast-data/go-vast-client/rest"
)

// Type aliases for easier imports and backward compatibility

type (
	// VMSConfig holds connection and authentication configuration for a VAST cluster.
	VMSConfig = core.VMSConfig

	// Params is a map[string]any for flexible API parameters.
	Params = core.Params

	// Record is a single API response as map[string]any.
	Record = core.Record

	// RecordSet is a collection of Record objects.
	RecordSet = core.RecordSet

	// Renderable is the interface for Record and RecordSet.
	Renderable = core.Renderable

	// DisplayableRecord is an interface for records that can be displayed.
	DisplayableRecord = core.DisplayableRecord

	// ApiError represents an error from the VAST API.
	ApiError = core.ApiError

	// TypedVMSRest is the strongly-typed client with compile-time type safety.
	TypedVMSRest = rest.TypedVMSRest

	// VMSRest is the default untyped client using map[string]any. Recommended for most use cases.
	VMSRest = rest.UntypedVMSRest

	// VastResourceAPI defines standard CRUD operations for VAST resources.
	VastResourceAPI = core.VastResourceAPI

	// VastResourceAPIWithContext extends VastResourceAPI with context support.
	VastResourceAPIWithContext = core.VastResourceAPIWithContext

	// InterceptableVastResourceAPI adds request/response interception to VastResourceAPIWithContext.
	InterceptableVastResourceAPI = core.InterceptableVastResourceAPI
)

// Error handling functions
var (
	// IsNotFoundErr checks if the error is a 404 Not Found error.
	IsNotFoundErr = core.IsNotFoundErr

	// IgnoreNotFound returns nil if the error is a 404, otherwise returns the original error.
	IgnoreNotFound = core.IgnoreNotFound

	// IgnoreStatusCodes returns nil if the error is an API error with one of the specified status codes.
	IgnoreStatusCodes = core.IgnoreStatusCodes

	// ExpectStatusCodes checks if the error is an API error with one of the specified status codes.
	ExpectStatusCodes = core.ExpectStatusCodes

	// IsApiError checks if the error is an ApiError.
	IsApiError = core.IsApiError
)

// NewTypedVMSRest creates a strongly-typed client with compile-time type safety.
// Use when you need strict API contracts and IDE auto-completion.
func NewTypedVMSRest(config *VMSConfig) (*TypedVMSRest, error) {
	return rest.NewTypedVMSRest(config)
}

// NewVMSRest creates the default untyped client using map[string]any for flexible resource handling.
// Recommended for most use cases. Adapts to API changes without regeneration.
func NewVMSRest(config *VMSConfig) (*VMSRest, error) {
	return rest.NewUntypedVMSRest(config)
}
