package core

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// Helper function for tests to simplify URL parsing
func must(u *url.URL, err error) *url.URL {
	if err != nil {
		panic(err)
	}
	return u
}

// Helper function to parse URL in tests
func parseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}

func expectQueryValue(t *testing.T, got string, key string, want string) {
	t.Helper()
	parsed, err := url.ParseQuery(got)
	if err != nil {
		t.Fatalf("failed to parse query: %v", err)
	}
	vals, ok := parsed[key]
	if !ok || len(vals) == 0 {
		t.Fatalf("key %q missing in query: %q", key, got)
	}
	if vals[0] != want {
		t.Fatalf("value for %q = %q, want %q (raw: %q)", key, vals[0], want, got)
	}
}

func TestConvertMapToQuery_Slices(t *testing.T) {
	// int slice
	q := convertMapToQuery(Params{"ids": []int{1, 2, 3}})
	expectQueryValue(t, q, "ids", "1,2,3")

	// int64 slice
	q = convertMapToQuery(Params{"ids": []int64{10, 20}})
	expectQueryValue(t, q, "ids", "10,20")

	// float64 slice
	q = convertMapToQuery(Params{"f": []float64{1.5, 2}})
	expectQueryValue(t, q, "f", "1.5,2")

	// float32 slice
	q = convertMapToQuery(Params{"f": []float32{3.25, 4}})
	// strconv formats like fmt.Sprint; normalize expected by converting to string explicitly
	wantF := strconv.FormatFloat(float64(3.25), 'f', -1, 64) + "," + strconv.FormatFloat(float64(4), 'f', -1, 64)
	expectQueryValue(t, q, "f", wantF)

	// string slice
	q = convertMapToQuery(Params{"names": []string{"alice", "bob"}})
	expectQueryValue(t, q, "names", "alice,bob")

	// bool slice
	q = convertMapToQuery(Params{"flags": []bool{true, false, true}})
	expectQueryValue(t, q, "flags", "true,false,true")

	// hetero slice
	q = convertMapToQuery(Params{"mix": []any{"x", 7, 2.5, false}})
	expectQueryValue(t, q, "mix", "x,7,2.5,false")

	// empty slice
	q = convertMapToQuery(Params{"empty": []int{}})
	expectQueryValue(t, q, "empty", "")
}

func TestConvertMapToQuery_Arrays(t *testing.T) {
	// int array
	q := convertMapToQuery(Params{"ids": [3]int{1, 2, 3}})
	expectQueryValue(t, q, "ids", "1,2,3")

	// string array
	q = convertMapToQuery(Params{"names": [2]string{"a", "b"}})
	expectQueryValue(t, q, "names", "a,b")

	// float64 array
	q = convertMapToQuery(Params{"f": [2]float64{1.25, 9}})
	expectQueryValue(t, q, "f", "1.25,9")

	// bool array
	q = convertMapToQuery(Params{"flags": [2]bool{false, true}})
	expectQueryValue(t, q, "flags", "false,true")
}

func TestValidateResponse(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		host       string
		port       uint64
		wantErrNil bool
	}{
		{
			name: "successful response 200",
			response: &http.Response{
				StatusCode: 200,
				Request: &http.Request{
					Method: "GET",
					URL:    must(parseURL("https://test.example.com/api")),
				},
			},
			host:       "test.example.com",
			port:       443,
			wantErrNil: true,
		},
		{
			name: "successful response 201",
			response: &http.Response{
				StatusCode: 201,
				Request: &http.Request{
					Method: "POST",
					URL:    must(parseURL("https://test.example.com/api")),
				},
			},
			host:       "test.example.com",
			port:       443,
			wantErrNil: true,
		},
		{
			name: "client error 400",
			response: &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(strings.NewReader("Bad Request")),
				Request: &http.Request{
					Method: "POST",
					URL:    must(parseURL("https://test.example.com/api")),
				},
			},
			host:       "test.example.com",
			port:       443,
			wantErrNil: false,
		},
		{
			name: "server error 500",
			response: &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
				Request: &http.Request{
					Method: "GET",
					URL:    must(parseURL("https://test.example.com/api")),
				},
			},
			host:       "test.example.com",
			port:       443,
			wantErrNil: false,
		},
		{
			name:       "nil response",
			response:   nil,
			host:       "test.example.com",
			port:       443,
			wantErrNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResponse(tt.response, tt.host, tt.port)
			if (err == nil) != tt.wantErrNil {
				t.Errorf("validateResponse() error = %v, wantErrNil %v", err, tt.wantErrNil)
			}

			if err != nil {
				var apiErr *ApiError
				if !IsApiError(err) {
					t.Errorf("validateResponse() error should be ApiError, got %T", err)
				} else {
					apiErr = err.(*ApiError)
					if tt.response != nil {
						if apiErr.StatusCode != tt.response.StatusCode {
							t.Errorf("ApiError.StatusCode = %v, want %v", apiErr.StatusCode, tt.response.StatusCode)
						}
					}
				}
			}
		})
	}
}

func TestPathToUrl(t *testing.T) {
	// Create a mock session
	mockSession := &mockRESTSession{
		config: &VMSConfig{
			Host:       "test.example.com",
			Port:       443,
			ApiVersion: "v5",
		},
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "relative path",
			input:   "/users",
			want:    "https://test.example.com:443/api/v5/users",
			wantErr: false,
		},
		{
			name:    "path without leading slash",
			input:   "users",
			want:    "https://test.example.com:443/api/v5/users",
			wantErr: false,
		},
		{
			name:    "path with query parameters",
			input:   "/users?name=test",
			want:    "https://test.example.com:443/api/v5/users?name=test",
			wantErr: false,
		},
		{
			name:    "full URL - should return unchanged",
			input:   "https://other.com/api/users",
			want:    "https://other.com/api/users",
			wantErr: false,
		},
		{
			name:    "http URL - should return unchanged",
			input:   "http://other.com/api/users",
			want:    "http://other.com/api/users",
			wantErr: false,
		},
		{
			name:    "complex path with encoded characters",
			input:   "/users with spaces",
			want:    "https://test.example.com:443/api/v5/users%20with%20spaces",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pathToUrl(mockSession, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("pathToUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("pathToUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildUrl(t *testing.T) {
	mockSession := &mockRESTSession{
		config: &VMSConfig{
			Host:       "test.example.com",
			Port:       8443,
			ApiVersion: "v5",
		},
	}

	tests := []struct {
		name    string
		path    string
		query   string
		apiVer  string
		want    string
		wantErr bool
	}{
		{
			name:    "simple path",
			path:    "users",
			query:   "",
			apiVer:  "",
			want:    "https://test.example.com:8443/api/v5/users/",
			wantErr: false,
		},
		{
			name:    "path with query",
			path:    "users",
			query:   "name=test&limit=10",
			apiVer:  "",
			want:    "https://test.example.com:8443/api/v5/users/?name=test&limit=10",
			wantErr: false,
		},
		{
			name:    "custom API version",
			path:    "users",
			query:   "",
			apiVer:  "v6",
			want:    "https://test.example.com:8443/api/v6/users/",
			wantErr: false,
		},
		{
			name:    "path with leading/trailing slashes",
			path:    "/users/",
			query:   "",
			apiVer:  "",
			want:    "https://test.example.com:8443/api/v5/users/",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			query:   "",
			apiVer:  "",
			want:    "https://test.example.com:8443/api/v5/",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildUrl(mockSession, tt.path, tt.query, tt.apiVer)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCheckVastResourceVersionCompat is tested in integration tests due to complex VastResourceAPI interface dependencies

func TestConvertMapToQuery(t *testing.T) {
	tests := []struct {
		name   string
		params Params
		want   []string // Multiple valid orderings
	}{
		{
			name:   "empty params",
			params: Params{},
			want:   []string{""},
		},
		{
			name:   "single param",
			params: Params{"name": "test"},
			want:   []string{"name=test"},
		},
		{
			name:   "multiple params",
			params: Params{"name": "test", "id": 123},
			want:   []string{"id=123&name=test", "name=test&id=123"}, // URL encoding can vary order
		},
		{
			name:   "param with special characters",
			params: Params{"query": "test value"},
			want:   []string{"query=test+value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMapToQuery(tt.params)

			// Check if result matches any of the expected possibilities
			found := false
			for _, expected := range tt.want {
				if got == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("convertMapToQuery() = %v, want one of %v", got, tt.want)
			}
		})
	}
}

func TestGetResponseBodyAsStr(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
		want     string
	}{
		{
			name:     "nil response",
			response: nil,
			want:     "",
		},
		{
			name: "valid JSON response",
			response: &http.Response{
				Body: io.NopCloser(strings.NewReader(`{"name":"test","id":123}`)),
			},
			want: "", // We'll test this differently since JSON key order varies
		},
		{
			name: "invalid JSON response",
			response: &http.Response{
				Body: io.NopCloser(strings.NewReader("not json")),
			},
			want: "not json",
		},
		{
			name: "empty response body",
			response: &http.Response{
				Body: io.NopCloser(strings.NewReader("")),
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getResponseBodyAsStr(tt.response)
			if tt.name == "valid JSON response" {
				// For JSON, just check that it's valid JSON with indentation
				if !strings.Contains(got, "\"name\"") || !strings.Contains(got, "\"id\"") {
					t.Errorf("getResponseBodyAsStr() should contain name and id fields, got %v", got)
				}
				if !strings.Contains(got, "  ") {
					t.Errorf("getResponseBodyAsStr() should be indented, got %v", got)
				}
			} else if got != tt.want {
				t.Errorf("getResponseBodyAsStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSanitizeVersion is removed - sanitizeVersion function was removed from codebase
// func TestSanitizeVersion(t *testing.T) {
// 	t.Skip("sanitizeVersion function was removed from the codebase")
// }

// Mock implementations for testing

type mockRESTSession struct {
	config *VMSConfig
}

func (m *mockRESTSession) Get(ctx context.Context, url string, params Params, headers []http.Header) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) Post(ctx context.Context, url string, params Params, headers []http.Header) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) Put(ctx context.Context, url string, params Params, headers []http.Header) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) Patch(ctx context.Context, url string, params Params, headers []http.Header) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) Delete(ctx context.Context, url string, params Params, headers []http.Header) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) GetConfig() *VMSConfig {
	return m.config
}

func (m *mockRESTSession) GetAuthenticator() Authenticator {
	return nil
}

// Tests for header consolidation and multipart detection

func TestConsolidateHeaders_NoCustomHeaders(t *testing.T) {
	session := &mockRESTSession{config: &VMSConfig{UserAgent: "TestAgent/1.0"}}

	result := consolidateHeaders(session, nil)

	// Check default headers are set
	if result.Get(HeaderAccept) != ContentTypeJSON {
		t.Errorf("Expected Accept header to be %s, got %s", ContentTypeJSON, result.Get(HeaderAccept))
	}

	if result.Get(HeaderContentType) != ContentTypeJSON {
		t.Errorf("Expected Content-Type header to be %s, got %s", ContentTypeJSON, result.Get(HeaderContentType))
	}

	if result.Get(HeaderUserAgent) != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent header to be TestAgent/1.0, got %s", result.Get(HeaderUserAgent))
	}
}

func TestConsolidateHeaders_CustomHeadersOverrideDefaults(t *testing.T) {
	session := &mockRESTSession{config: &VMSConfig{UserAgent: "TestAgent/1.0"}}

	customHeaders := []http.Header{{
		HeaderContentType: []string{ContentTypeMultipartForm},
		HeaderAccept:      []string{ContentTypeTextPlain},
		"X-Custom":        []string{"CustomValue"},
	}}

	result := consolidateHeaders(session, customHeaders)

	// Check custom headers override defaults
	if result.Get(HeaderContentType) != ContentTypeMultipartForm {
		t.Errorf("Expected Content-Type to be overridden to %s, got %s", ContentTypeMultipartForm, result.Get(HeaderContentType))
	}

	if result.Get(HeaderAccept) != ContentTypeTextPlain {
		t.Errorf("Expected Accept to be overridden to %s, got %s", ContentTypeTextPlain, result.Get(HeaderAccept))
	}

	// Check default is still applied for non-overridden headers
	if result.Get(HeaderUserAgent) != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent default to be preserved, got %s", result.Get(HeaderUserAgent))
	}

	// Check custom header is preserved
	if result.Get("X-Custom") != "CustomValue" {
		t.Errorf("Expected X-Custom header to be CustomValue, got %s", result.Get("X-Custom"))
	}
}

func TestMultipartDetection_MultipartFormData(t *testing.T) {
	session := &mockRESTSession{config: &VMSConfig{UserAgent: "TestAgent/1.0"}}

	customHeaders := []http.Header{{
		HeaderContentType: []string{ContentTypeMultipartForm},
	}}

	result := consolidateHeaders(session, customHeaders)
	contentType := result.Get(HeaderContentType)
	useMultipart := strings.Contains(strings.ToLower(contentType), ContentTypeMultipartForm)

	if !useMultipart {
		t.Errorf("Expected multipart to be detected for Content-Type: %s", contentType)
	}
}

func TestMultipartDetection_NonMultipart(t *testing.T) {
	session := &mockRESTSession{config: &VMSConfig{UserAgent: "TestAgent/1.0"}}

	customHeaders := []http.Header{{
		HeaderContentType: []string{ContentTypeJSON},
	}}

	result := consolidateHeaders(session, customHeaders)
	contentType := result.Get(HeaderContentType)
	useMultipart := strings.Contains(strings.ToLower(contentType), ContentTypeMultipartForm)

	if useMultipart {
		t.Errorf("Expected multipart NOT to be detected for Content-Type: %s", contentType)
	}
}

func TestMultipartDetection_WithBoundary(t *testing.T) {
	session := &mockRESTSession{config: &VMSConfig{UserAgent: "TestAgent/1.0"}}

	customHeaders := []http.Header{{
		HeaderContentType: []string{"multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW"},
	}}

	result := consolidateHeaders(session, customHeaders)
	contentType := result.Get(HeaderContentType)
	useMultipart := strings.Contains(strings.ToLower(contentType), ContentTypeMultipartForm)

	if !useMultipart {
		t.Errorf("Expected multipart to be detected for Content-Type with boundary: %s", contentType)
	}
}

func TestConsolidateHeaders_MultipleHeaderBlocks(t *testing.T) {
	session := &mockRESTSession{config: &VMSConfig{UserAgent: "TestAgent/1.0"}}

	customHeaders := []http.Header{
		{
			HeaderAccept: []string{ContentTypeJSON},
			"X-First":    []string{"FirstValue"},
		},
		{
			HeaderContentType: []string{ContentTypeMultipartForm},
			"X-Second":        []string{"SecondValue"},
		},
	}

	result := consolidateHeaders(session, customHeaders)

	// Check all custom headers are applied
	if result.Get(HeaderAccept) != ContentTypeJSON {
		t.Errorf("Expected Accept to be %s, got %s", ContentTypeJSON, result.Get(HeaderAccept))
	}

	if result.Get(HeaderContentType) != ContentTypeMultipartForm {
		t.Errorf("Expected Content-Type to be %s, got %s", ContentTypeMultipartForm, result.Get(HeaderContentType))
	}

	if result.Get("X-First") != "FirstValue" {
		t.Errorf("Expected X-First to be FirstValue, got %s", result.Get("X-First"))
	}

	if result.Get("X-Second") != "SecondValue" {
		t.Errorf("Expected X-Second to be SecondValue, got %s", result.Get("X-Second"))
	}

	// Default should still be applied
	if result.Get(HeaderUserAgent) != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent default to be TestAgent/1.0, got %s", result.Get(HeaderUserAgent))
	}
}

func TestConsolidateHeaders_EmptyCustomHeaders(t *testing.T) {
	session := &mockRESTSession{config: &VMSConfig{UserAgent: "TestAgent/1.0"}}

	// Empty slice should behave same as nil
	customHeaders := []http.Header{}

	result := consolidateHeaders(session, customHeaders)

	// Should get all defaults
	if result.Get(HeaderAccept) != ContentTypeJSON {
		t.Errorf("Expected Accept header to be %s, got %s", ContentTypeJSON, result.Get(HeaderAccept))
	}

	if result.Get(HeaderContentType) != ContentTypeJSON {
		t.Errorf("Expected Content-Type header to be %s, got %s", ContentTypeJSON, result.Get(HeaderContentType))
	}

	if result.Get(HeaderUserAgent) != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent header to be TestAgent/1.0, got %s", result.Get(HeaderUserAgent))
	}
}
