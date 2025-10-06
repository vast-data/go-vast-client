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

// Operations represents the unified ops marker configuration
// Replaces the combination of Details + Upsert markers
// Supports any combination of CRUD operations
// Usage: +apityped:ops:CRUD=users
// Usage: +apityped:ops:R=versions (read-only)
// Usage: +apityped:ops:CU=certificates (create + update, no delete)
// Usage: +apityped:ops:UD=configs (update + delete, no create)
type Operations struct {
	Operations string // Combination of C, L, R, U, D (e.g., "CLRUD", "LR", "CU", "UD")
	URL        string // Resource path like "users", "versions", etc.
}

// HasCreate returns true if operations include Create
func (o Operations) HasCreate() bool {
	return containsChar(o.Operations, 'C')
}

// HasList returns true if operations include List (getting multiple resources or one by filter)
func (o Operations) HasList() bool {
	return containsChar(o.Operations, 'L')
}

// HasRead returns true if operations include Read (getting a single resource by ID)
func (o Operations) HasRead() bool {
	return containsChar(o.Operations, 'R')
}

// HasUpdate returns true if operations include Update
func (o Operations) HasUpdate() bool {
	return containsChar(o.Operations, 'U')
}

// HasDelete returns true if operations include Delete
func (o Operations) HasDelete() bool {
	return containsChar(o.Operations, 'D')
}

// ExtraMethod represents an extra method marker configuration
// Generates: Custom typed/untyped methods for specific endpoints
// Usage: +apityped:extraMethod:GET=/users/{id}/tenant_data/
// Usage: +apiuntyped:extraMethod:PATCH=/users/{id}/tenant_data/
// Usage: +apiall:extraMethod:GET=/users/{id}/tenant_data/ (generates both typed and untyped)
type ExtraMethod struct {
	Method string // HTTP method: GET, POST, PUT, PATCH, DELETE, etc.
	Path   string // Full path with placeholders like "/users/{id}/tenant_data/"
}

// containsChar checks if a string contains a specific character
func containsChar(s string, c rune) bool {
	for _, char := range s {
		if char == c {
			return true
		}
	}
	return false
}
