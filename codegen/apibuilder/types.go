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

// Details represents a details marker configuration
// Generates: SearchParams (from query params) + DetailsModel (from response)
// Usage: +apityped:details:GET=apitokens
// Usage: +apityped:details:PATCH=quotas
type Details struct {
	Method string // GET or PATCH
	URL    string // Resource path like "apitokens" or "quotas"
}

// Upsert represents a create/update marker configuration
// Generates: RequestBody (from request body) + UpsertModel (from response)
// Usage: +apityped:upsert:POST=apitokens
// Usage: +apityped:upsert:PUT=quotas
// Usage: +apityped:upsert:PATCH=quotas
type Upsert struct {
	Method string // POST, PUT, or PATCH
	URL    string // Resource path like "apitokens" or "quotas"
}

// DetailsQuery represents a details query marker configuration (DEPRECATED - use Details)
// Usage: +apityped:detailsQuery:GET=apitokens
type DetailsQuery struct {
	Method string
	URL    string
}

// UpsertQuery represents a create/update query marker configuration (DEPRECATED - use Upsert)
// Usage: +apityped:upsertQuery:POST=apitokens
type UpsertQuery struct {
	Method string
	URL    string
}

// SearchQuery represents a search query marker configuration (DEPRECATED - use DetailsQuery)
// Usage: +apityped:searchQuery:GET=apitokens
type SearchQuery struct {
	Method string
	URL    string
}

// CreateQuery represents a create/update query marker configuration (DEPRECATED - use UpsertQuery)
// Usage: +apityped:createQuery:POST=apitokens
type CreateQuery struct {
	Method string
	URL    string
}

// RequestBody represents a request body marker configuration (DEPRECATED - use UpsertQuery)
// Usage: +apityped:requestBody:POST=quotas
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

// APITyped holds the complete API configuration for code generation
type APITyped struct {
	Endpoints map[string]*APIEndpoint
}
