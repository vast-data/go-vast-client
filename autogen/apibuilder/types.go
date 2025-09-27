package apibuilder

// RequestURL represents a request URL marker configuration
// Usage: +apibuilder:requestUrl:GET:/api/users
// Usage: +apibuilder:requestUrl:POST:/api/users
type RequestURL struct {
	Method string
	URL    string
}

// ResponseURL represents a response URL marker configuration
// Usage: +apibuilder:responseUrl:GET:/api/users
// Usage: +apibuilder:responseUrl:DELETE:/api/users/{id}
type ResponseURL struct {
	Method string
	URL    string
}

// SearchQuery represents a search query marker configuration
// Usage: +apibuilder:searchQuery:GET=quotas
// Usage: +apibuilder:searchQuery:SCHEMA=Quota
type SearchQuery struct {
	Method string
	URL    string
}

// RequestBody represents a request body marker configuration
// Usage: +apibuilder:requestBody:POST=quotas
// Usage: +apibuilder:requestBody:SCHEMA=Quota
type RequestBody struct {
	Method string
	URL    string
}

// ResponseBody represents a response body marker configuration
// Usage: +apibuilder:responseBody:POST=quotas
// Usage: +apibuilder:responseBody:SCHEMA=Quota
type ResponseBody struct {
	Method string
	URL    string
}

// RequestModel represents a request model marker
// Usage: +apibuilder:requestModel:UserCreateRequest
// Usage: +apibuilder:requestModel:ProductUpdateRequest
type RequestModel struct {
	Model string `marker:"model"`
}

// ResponseModel represents a response model marker
// Usage: +apibuilder:responseModel:UserResponse
// Usage: +apibuilder:responseModel:ErrorResponse
type ResponseModel struct {
	Model string `marker:"model"`
}

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
