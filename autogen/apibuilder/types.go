package apibuilder

// RequestURL represents a request URL marker configuration
// Usage: +apityped:requestUrl:GET:/api/users
// Usage: +apityped:requestUrl:POST:/api/users
type RequestURL struct {
	Method string
	URL    string
}

// ResponseURL represents a response URL marker configuration
// Usage: +apityped:responseUrl:GET:/api/users
// Usage: +apityped:responseUrl:DELETE:/api/users/{id}
type ResponseURL struct {
	Method string
	URL    string
}

// SearchQuery represents a search query marker configuration
// Usage: +apityped:searchQuery:GET=quotas
// Usage: +apityped:searchQuery:SCHEMA=Quota
type SearchQuery struct {
	Method string
	URL    string
}

// RequestBody represents a request body marker configuration
// Usage: +apityped:requestBody:POST=quotas
// Usage: +apityped:requestBody:SCHEMA=Quota
type RequestBody struct {
	Method string
	URL    string
}

// ResponseBody represents a response body marker configuration
// Usage: +apityped:responseBody:POST=quotas
// Usage: +apityped:responseBody:SCHEMA=Quota
type ResponseBody struct {
	Method string
	URL    string
}

// RequestModel represents a request model marker
// Usage: +apityped:requestModel:UserCreateRequest
// Usage: +apityped:requestModel:ProductUpdateRequest
type RequestModel struct {
	Model string `marker:"model"`
}

// ResponseModel represents a response model marker
// Usage: +apityped:responseModel:UserResponse
// Usage: +apityped:responseModel:ErrorResponse
type ResponseModel struct {
	Model string `marker:"model"`
}

// ReadOnly represents a read-only resource marker
// Usage: +apityped:readOnly
type ReadOnly struct{}

// APIEndpoint represents a complete API endpoint configuration
// This can be used to collect all related markers for an endpoint
type APIEndpoint struct {
	RequestURL    *RequestURL
	ResponseURL   *ResponseURL
	RequestModel  *RequestModel
	ResponseModel *ResponseModel
}

// APIBuilder holds the complete API configuration for code generation
type APIBuilder struct {
	Endpoints map[string]*APIEndpoint
}
