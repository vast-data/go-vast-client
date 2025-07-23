package vast_client

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

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

func TestSanitizeVersion(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		wantVersion   string
		wantTruncated bool
	}{
		{
			name:          "core version only",
			version:       "5.3.0",
			wantVersion:   "5.3.0",
			wantTruncated: false,
		},
		{
			name:          "version with build info",
			version:       "5.3.0.1234.abcd",
			wantVersion:   "5.3.0",
			wantTruncated: true,
		},
		{
			name:          "version with pre-release",
			version:       "5.3.0-beta.1",
			wantVersion:   "5.3.0-beta.1",
			wantTruncated: false,
		},
		{
			name:          "short version",
			version:       "5.3",
			wantVersion:   "5.3",
			wantTruncated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotTruncated := sanitizeVersion(tt.version)
			if gotVersion != tt.wantVersion {
				t.Errorf("sanitizeVersion() version = %v, want %v", gotVersion, tt.wantVersion)
			}
			if gotTruncated != tt.wantTruncated {
				t.Errorf("sanitizeVersion() truncated = %v, want %v", gotTruncated, tt.wantTruncated)
			}
		})
	}
}

// Mock implementations for testing

type mockRESTSession struct {
	config *VMSConfig
}

func (m *mockRESTSession) Get(context.Context, string, Params) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) Post(context.Context, string, Params) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) Put(context.Context, string, Params) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) Patch(context.Context, string, Params) (Renderable, error) {
	return Record{}, nil
}

func (m *mockRESTSession) Delete(context.Context, string, Params) (Renderable, error) {
	return EmptyRecord{}, nil
}

func (m *mockRESTSession) GetConfig() *VMSConfig {
	return m.config
}

func (m *mockRESTSession) GetAuthenticator() Authenticator {
	return nil
}

// Helper function for tests
func parseURL(urlStr string) (*url.URL, error) {
	return url.Parse(urlStr)
}
