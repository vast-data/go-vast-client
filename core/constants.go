package core

// HTTP-related constants for REST operations
// These constants provide type-safe header names, content types, and auth types

// HTTP Header Names
const (
	HeaderAccept        = "Accept"
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderContentLength = "Content-Length"
	HeaderUserAgent     = "User-Agent"
	HeaderXTenantName   = "X-Tenant-Name"
)

// HTTP Content Types
const (
	ContentTypeJSON           = "application/json"
	ContentTypeMultipartForm  = "multipart/form-data"
	ContentTypeOpenAPI        = "application/openapi+json"
	ContentTypeFormURLEncoded = "application/x-www-form-urlencoded"
	ContentTypeTextPlain      = "text/plain"
	ContentTypeOctetStream    = "application/octet-stream"
)

// HTTP Authentication Types
const (
	AuthTypeBasic  = "Basic"
	AuthTypeBearer = "Bearer"
)
