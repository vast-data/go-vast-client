package vast_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"strings"
)

// validateResponse checks the response for valid HTTP status codes (specifically for 2xx codes).
// It returns an error if the status code is not a valid 2xx code or if the response is nil.
//
// Arguments:
// - response: the HTTP response to validate
// - err: the error to check (if any)
//
// Returns:
// - error: an error if validation fails
func validateResponse(response *http.Response) error {
	requestURL := "<unknown URL>"
	method := "<unknown method>"
	if response == nil {
		return &ApiError{
			Method:     method,
			URL:        requestURL,
			StatusCode: 0,
			Body:       "server unreachable: verify the host is correct and the network is accessible",
		}
	}
	if response.StatusCode >= 200 && response.StatusCode <= 299 {
		return nil
	}
	if response.Request != nil {
		if response.Request.URL != nil {
			requestURL = response.Request.URL.String()
		}
		method = response.Request.Method
	}
	return &ApiError{
		Method:     method,
		URL:        requestURL,
		StatusCode: response.StatusCode,
		Body:       getResponseBodyAsStr(response),
	}
}

// pathToUrl returns a full URI string based on the provided input.
// If the input string is already a full URI (i.e., contains a scheme like "http" or "https"),
// it is returned unchanged.
// Otherwise, the function constructs a full URI using the session's configuration,
// appending the input path (with optional query parameters) to the base API path.
func pathToUrl(s RESTSession, input string) (string, error) {
	parsedURL, parseErr := urlpkg.Parse(input)
	if parseErr == nil && parsedURL.Scheme != "" {
		return input, nil // already a full URI
	}
	// Ensure input starts with a slash
	if !strings.HasPrefix(input, "/") {
		input = "/" + input
	}
	config := s.GetConfig()

	// Now it's a valid request URI
	pathAndQuery, err := urlpkg.ParseRequestURI(input)
	if err != nil {
		return "", fmt.Errorf("invalid relative URL: %w", err)
	}
	joinedPath, err := urlpkg.JoinPath("api", config.ApiVersion, strings.Trim(pathAndQuery.Path, "/"))
	if err != nil {
		return "", err
	}
	fullURL := &urlpkg.URL{
		Scheme:   "https",
		Host:     fmt.Sprintf("%s:%v", config.Host, config.Port),
		Path:     joinedPath,
		RawQuery: pathAndQuery.RawQuery,
	}
	return fullURL.String(), nil
}

func buildUrl(s RESTSession, path, query, apiVer string) (string, error) {
	var err error
	config := s.GetConfig()
	if apiVer == "" {
		apiVer = config.ApiVersion
	}
	if path, err = urlpkg.JoinPath("api", apiVer, strings.Trim(path, "/")); err != nil {
		return "", err
	}
	url := urlpkg.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s:%v", config.Host, config.Port),
		Path:   path,
	}
	if query != "" {
		url.RawQuery = query
	}
	return url.String(), nil
}

// Check if current VAST cluster version support triggered API
func checkVastResourceVersionCompat(ctx context.Context, r VastResourceAPI) error {
	resourceType := r.GetResourceType()
	availableFromVersion := r.getAvailableFromVersion()
	rest := r.getRest()
	if availableFromVersion == nil {
		return nil
	}
	compareOrd, err := rest.Versions.CompareWithWithContext(ctx, availableFromVersion)
	if err != nil {
		return err
	}
	clusterVersion, _ := rest.Versions.GetVersionWithContext(ctx)
	if compareOrd == -1 {
		return fmt.Errorf("resource %q is not supported in VAST cluster version %s (supported from version %s)", resourceType, clusterVersion, availableFromVersion)
	}
	return nil
}

// convertMapToQuery converts a map[string]any to a URL query string.
// Values are stringified using fmt.Sprint.
func convertMapToQuery(params Params) string {
	values := urlpkg.Values{}
	for k, v := range params {
		values.Set(k, fmt.Sprint(v))
	}
	return values.Encode()
}

// getResponseBodyAsStr reads and returns the HTTP response body as a string.
// If the response body contains valid JSON, it returns a pretty-printed version.
// If the JSON indentation fails or the body is not JSON, it returns the raw body as a string.
// If the response is nil or an error occurs during reading, it returns an empty string.
//
// Note: This function consumes and closes the response body.
func getResponseBodyAsStr(r *http.Response) string {
	var b bytes.Buffer
	if r == nil {
		return ""
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ""
	}
	//Let's try to make it a pretty json if not we will just dump the body
	err = json.Indent(&b, body, "", "  ")
	if err == nil {
		return string(b.Bytes())
	}
	return string(body)
}

// sanitizeVersion truncates all segments of Cluster Version above core (x.y.z)
func sanitizeVersion(version string) (string, bool) {
	segments := strings.Split(version, ".")
	truncated := len(segments) > 3
	return strings.Join(segments[:3], "."), truncated
}
