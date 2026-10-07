package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"
)

// VMSConfig represents the configuration required to create a VMS session.
type VMSConfig struct {
	Host         string // The hostname or IP address of the VMS API server.
	Port         uint64 // The port to connect to on the VMS API server.
	Username     string // The username for authentication (used with Password).
	Password     string // The password for authentication (used with Username).
	ApiToken     string // Optional API token for authentication (alternative to Username/Password).
	UseBasicAuth bool   // If true, use HTTP Basic Authentication instead of JWT (requires Username/Password).
	Tenant       string // Optional tenant name. Default JWT login uses POST /api/token/ with X-Tenant-Name.
	// UseTenantTokenPath, when true with Tenant, obtains the JWT via POST /api/token/{tenant}/
	// instead of sending X-Tenant-Name on /api/token/. VMS treats path and header as equivalent
	// at login.
	UseTenantTokenPath bool
	SslVerify          bool           // Whether to verify SSL certificates.
	RespectProxy       bool           // Whether to respect proxy environment variables (HTTP_PROXY, HTTPS_PROXY, NO_PROXY).
	Timeout            *time.Duration // HTTP client timeout. If nil, a default is applied by validators.
	MaxConnections     int            // Maximum number of concurrent HTTP connections.
	UserAgent          string         // Optional custom User-Agent header to use in HTTP requests. If empty, a default may be applied.
	ApiVersion         string         // Optional API version
	PageSize           int            // Default page size for iterators
	// Context is an optional external context for controlling HTTP request lifecycle.
	// When provided, it will be used as the parent context for all HTTP requests made by the client.
	Context context.Context

	// BeforeRequestFn is an optional function hook executed before an API request is sent.
	// It allows for request inspection, mutation, or logging.
	//
	// Parameters:
	//   - ctx: The request context for managing deadlines and cancellations.
	//   - req: Request object
	//   - verb: The HTTP method (e.g., GET, POST, PUT).
	//   - url: The target URL (path and query parameters).
	//   - body: The request body reader, typically containing JSON payload.
	//
	// Return:
	//   - error: Any error returned will abort the request.
	BeforeRequestFn func(ctx context.Context, r *http.Request, verb, url string, body io.Reader) error

	// AfterRequestFn is an optional function hook executed after every HTTP attempt
	// that produced a response (or a transport failure). It runs for 2xx and non-2xx
	// statuses alike, so hooks can log statusCode even when the request fails.
	//
	// Parameters:
	//   - ctx: The request context for managing deadlines and cancellations.
	//   - response: Parsed body on success (Record / RecordSet). nil means the
	//     request failed (non-2xx or transport error) — do not treat nil as an
	//     empty successful body; empty Record{} is a successful empty response.
	//   - statusCode: HTTP status code (2xx on success, non-2xx on API errors, 0
	//     when there was no HTTP response / transport error).
	//
	// Returns:
	//   - A potentially modified Renderable object (used only on success paths).
	//   - An error, if processing the response fails (takes precedence over ApiError).
	AfterRequestFn func(ctx context.Context, response Renderable, statusCode int) (Renderable, error)

	// FillFn optionally overrides the default function used to populate structs
	// from generic Record maps. If provided, this function is invoked instead of
	// the default JSON-based marshal/unmarshal logic.
	//
	// This is useful for customizing how API responses are decoded into typed
	// structures — for example, using a different decoding library or adding hooks.
	//
	// Parameters:
	//   - r: The Record to fill from (typically parsed from JSON response).
	//   - container: A pointer to a struct to be populated.
	//
	// Returns:
	//   - error: Any decoding or validation error encountered during population.
	FillFn func(r Record, container any) error
}

// VMSConfigFunc defines a function that can modify or validate a VMSConfig.
type VMSConfigFunc func(*VMSConfig) error

// Validate applies the given VMSConfigFunc validators to the config.
// Returns the first error encountered, or nil if all validators pass.
func (config *VMSConfig) Validate(validators ...VMSConfigFunc) error {
	for _, fn := range validators {
		if err := fn(config); err != nil {
			return err
		}
	}
	return nil
}

// WithTimeout returns a VMSConfigFunc that sets a default timeout if none is provided.
func WithTimeout(timeout time.Duration) VMSConfigFunc {
	return func(config *VMSConfig) error {
		if config.Timeout == nil {
			config.Timeout = &timeout
		}
		return nil
	}
}

// WithMaxConnections returns a VMSConfigFunc that sets the maximum number of connections
// if not explicitly provided.
func WithMaxConnections(maxConnections int) VMSConfigFunc {
	return func(config *VMSConfig) error {
		if config.MaxConnections == 0 {
			config.MaxConnections = maxConnections
		}
		return nil
	}
}

// WithHost validates that the Host field is not empty.
// Returns an error if Host is an empty string.
func WithHost(config *VMSConfig) error {
	if config.Host == "" {
		return errors.New("host cannot be empty string")
	}
	return nil
}

// WithPort returns a VMSConfigFunc that sets a default port if none is provided.
func WithPort(defaultPort uint64) VMSConfigFunc {
	return func(config *VMSConfig) error {
		if config.Port == 0 {
			config.Port = defaultPort
		}
		return nil
	}
}

// WithAuth validates that either a username/password combination or an API token
// is provided for authentication. Returns an error if neither is set.
func WithAuth(config *VMSConfig) error {
	hasUserPass := config.Username != "" && config.Password != ""
	hasToken := config.ApiToken != ""
	if !hasUserPass && !hasToken {
		return errors.New("either username/password or api token must be provided")
	}
	return nil
}

// WithUserAgent sets a default User-Agent header if none is provided in the config.
// This helps identify the client in HTTP requests. If UserAgent is empty,
// it defaults to "VASTData Client".
func WithUserAgent(config *VMSConfig) error {
	if config.UserAgent == "" {
		config.UserAgent = fmt.Sprintf(
			"%s,os:%s,arch:%s",
			fmt.Sprintf("vast-go-client-%s", ClientVersion()),
			runtime.GOOS,
			runtime.GOARCH,
		)
	}
	return nil
}

// WithApiVersion sets a default API version
// NOTE: API version can be overwritten for particular VastResource
func WithApiVersion(defaultVer string) VMSConfigFunc {
	return func(config *VMSConfig) error {
		if config.ApiVersion == "" {
			config.ApiVersion = defaultVer
		}
		return nil
	}
}

// WithFillFn is a VMSConfigFunc that installs a custom FillFn into the global
// fillFunc used by the Record.Fill method.
//
// This allows the client to globally override the default record-to-struct
// population logic.
func WithFillFn(config *VMSConfig) error {
	if config.FillFn != nil {
		fillFunc = config.FillFn
	}
	return nil
}
