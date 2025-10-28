package core

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type contextKey string

const (
	caller     contextKey = "@caller" // VastResource Caller object key
	maxRetries int        = 3
)

type RESTSession interface {
	Get(context.Context, string, Params, []http.Header) (Renderable, error)
	Post(context.Context, string, Params, []http.Header) (Renderable, error)
	Put(context.Context, string, Params, []http.Header) (Renderable, error)
	Patch(context.Context, string, Params, []http.Header) (Renderable, error)
	Delete(context.Context, string, Params, []http.Header) (Renderable, error)
	GetConfig() *VMSConfig
	GetAuthenticator() Authenticator
}

// ApiError represents an error returned from an API request.
type ApiError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
	hints      string
}

// Error implements the error interface.
func (e *ApiError) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("response body: %s", e.Body)
	}
	if e.hints == "" {
		return fmt.Sprintf(
			"%s request to %s returned status code %d"+
				" — response body: %s", e.Method, e.URL, e.StatusCode, e.Body,
		)
	} else {
		return fmt.Sprintf(
			"%s request to %s returned status code %d"+
				" — response body: %s\nResource details:\n%s", e.Method, e.URL, e.StatusCode, e.Body, e.hints,
		)
	}

}

func IsApiError(err error) bool {
	var apiErr *ApiError
	return errors.As(err, &apiErr)
}

func IgnoreStatusCodes(err error, codes ...int) error {
	if !IsApiError(err) {
		return err
	}
	apiErr := err.(*ApiError)
	for _, code := range codes {
		if apiErr.StatusCode == code {
			return nil
		}
	}
	return err
}

func ExpectStatusCodes(err error, codes ...int) bool {
	if !IsApiError(err) {
		return false
	}
	found := false
	apiErr := err.(*ApiError)
	for _, code := range codes {
		if apiErr.StatusCode == code {
			found = true
			break
		}
	}
	return found
}

type VMSSession struct {
	config *VMSConfig
	client *http.Client
	auth   Authenticator
}

type VMSSessionMethod func(context.Context, string, Params, []http.Header) (Renderable, error)

func NewVMSSession(config *VMSConfig) (*VMSSession, error) {
	//Create a new session object
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: !config.SslVerify}
	transport.MaxConnsPerHost = config.MaxConnections
	transport.IdleConnTimeout = *config.Timeout
	client := &http.Client{Transport: transport}
	authenticator, err := createAuthenticator(config)
	if err != nil {
		return nil, err
	}
	session := &VMSSession{
		config: config,
		client: client,
		auth:   authenticator,
	}
	return session, nil
}

func Request[T RecordUnion](
	ctx context.Context,
	r VastResourceAPIWithContext,
	verb, path string,
	params, body Params,
) (T, error) {
	return RequestWithHeaders[T](ctx, r, verb, path, params, body, nil)
}

func RequestWithHeaders[T RecordUnion](
	ctx context.Context,
	r VastResourceAPIWithContext,
	verb, path string,
	params, body Params,
	headers []http.Header,
) (T, error) {
	var (
		vmsMethod VMSSessionMethod
		query     string
		err       error
	)
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, caller, r)
	verb = strings.ToUpper(verb)
	session := r.Session()

	switch verb {
	case http.MethodGet:
		vmsMethod = session.Get
	case http.MethodPost:
		vmsMethod = session.Post
	case http.MethodPut:
		vmsMethod = session.Put
	case http.MethodPatch:
		vmsMethod = session.Patch
	case http.MethodDelete:
		vmsMethod = session.Delete
	default:
		return nil, fmt.Errorf("unknown verb: %s", verb)
	}
	if params != nil {
		query = params.ToQuery()
	}
	url, err := buildUrl(session, path, query, session.GetConfig().ApiVersion)
	if err != nil {
		return nil, err
	}

	response, err := vmsMethod(ctx, url, body, headers)
	if err != nil {
		return nil, err
	}

	if typeMatch[Record](response) {
		// Some resources return single record
		// although query typically return list for others (for instance NonLocalUser)
		// We want to eliminate this discrepancy by casting Record to RecordSet
		var zero T
		if typeMatch[RecordSet](Renderable(zero)) {
			if !response.(Record).Empty() {
				response = RecordSet{response.(Record)}
			} else {
				response = RecordSet{}
			}
		}
	}

	resultVal, ok := response.(T)
	if !ok {
		return nil, fmt.Errorf(
			"unexpected response type for request to %s: got %T, expected %T — "+
				"consider converting the response to the expected type inside the doAfterRequest interceptor",
			url,
			response,
			*new(T),
		)
	}
	return resultVal, nil
}

func (s *VMSSession) Get(ctx context.Context, url string, _ Params, headers []http.Header) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodGet, url, nil, headers)
}

func (s *VMSSession) Post(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodPost, url, body, headers)
}

func (s *VMSSession) Put(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodPut, url, body, headers)
}

func (s *VMSSession) Patch(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodPatch, url, body, headers)
}

func (s *VMSSession) Delete(ctx context.Context, url string, body Params, headers []http.Header) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodDelete, url, body, headers)
}

// fetchSchema retrieves the OpenAPI schema using Basic Auth and custom headers
func (s *VMSSession) fetchSchema(ctx context.Context) (Renderable, error) {
	url, err := buildUrl(s, "", "", s.config.ApiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL for OpenAPI schema: %w", err)
	}
	// Basic Auth
	authStr := s.config.Username + ":" + s.config.Password
	encoded := base64.StdEncoding.EncodeToString([]byte(authStr))
	headers := []http.Header{{
		HeaderAuthorization: []string{AuthTypeBasic + " " + encoded},
		HeaderAccept:        []string{ContentTypeOpenAPI},
	}}
	return doRequest(ctx, s, http.MethodGet, url+"?format=openapi", nil, headers)
}

func (s *VMSSession) GetConfig() *VMSConfig {
	return s.config
}
func (s *VMSSession) GetAuthenticator() Authenticator {
	return s.auth
}

func consolidateHeaders(s RESTSession, customHeaders []http.Header) http.Header {
	finalHeaders := make(http.Header)

	// Apply custom headers first
	for _, header := range customHeaders {
		for key, values := range header {
			for _, value := range values {
				finalHeaders.Add(key, value)
			}
		}
	}

	// Set default headers only if not already provided
	if finalHeaders.Get(HeaderAccept) == "" {
		finalHeaders.Set(HeaderAccept, ContentTypeJSON)
	}

	if finalHeaders.Get(HeaderContentType) == "" {
		finalHeaders.Set(HeaderContentType, ContentTypeJSON)
	}

	if finalHeaders.Get(HeaderUserAgent) == "" {
		finalHeaders.Set(HeaderUserAgent, s.GetConfig().UserAgent)
	}

	return finalHeaders
}

func setupHeaders(s RESTSession, r *http.Request, headers http.Header) error {
	// Always set authentication headers
	s.GetAuthenticator().setAuthHeader(&r.Header)

	// Apply all consolidated headers in one pass
	for key, values := range headers {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}

	return nil
}

// doRequest Create and process the new HTTP request using the context
func doRequest(ctx context.Context, s *VMSSession, verb, url string, body Params, headers []http.Header) (Renderable, error) {
	// callerExist if request is processed via "request" method
	var (
		config            = s.GetConfig()
		resourceCaller    InterceptableVastResourceAPI
		requestData       io.Reader
		beforeRequestData io.Reader
		err               error
	)
	originResource, resourceExist := ctx.Value(caller).(InterceptableVastResourceAPI)
	if !resourceExist {
		resourceCaller = NewDummy(ctx, s)
	} else {
		resourceCaller = originResource
	}
	// Convert to full URI if needed.
	if url, err = pathToUrl(s, url); err != nil {
		return nil, err
	}

	// Consolidate headers
	finalHeaders := consolidateHeaders(s, headers)

	// Determine if multipart/form-data is being used
	contentType := finalHeaders.Get(HeaderContentType)
	useMultipart := strings.Contains(strings.ToLower(contentType), ContentTypeMultipartForm)

	if body == nil {
		requestData = bytes.NewReader(nil)
	} else {
		if useMultipart {
			// Use multipart form data
			multipartData, err := body.ToMultipartFormData()
			if err != nil {
				return nil, fmt.Errorf("failed to create multipart form data: %w", err)
			}
			requestData = multipartData.Body

			// Update the Content-Type header with the proper boundary
			finalHeaders.Set(HeaderContentType, multipartData.ContentType)
		} else {
			// Use regular JSON body
			if requestData, err = body.ToBody(); err != nil {
				return nil, err
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, verb, url, requestData)
	if err != nil {
		return nil, err
	}
	// Prepare beforeRequestData for interceptors
	if body != nil {
		if useMultipart {
			// For multipart data, create a fresh copy for the interceptor
			multipartData, err := body.ToMultipartFormData()
			if err != nil {
				return nil, fmt.Errorf("failed to create multipart form data for interceptor: %w", err)
			}
			beforeRequestData = multipartData.Body
		} else {
			if beforeRequestData, err = body.ToBody(); err != nil {
				return nil, err
			}
		}
	}
	// Setup headers (both custom and defaults)
	if err = setupHeaders(s, req, finalHeaders); err != nil {
		return nil, err
	}

	// before request interceptor
	if err = resourceCaller.doBeforeRequest(ctx, req, verb, url, beforeRequestData); err != nil {
		return nil, err
	}
	response, responseErr := s.client.Do(req)

	if responseErr != nil {
		return nil, fmt.Errorf("failed to perform %s request to %s, error %v", verb, url, responseErr)
	}
	if err = validateResponse(response, config.Host, config.Port); err != nil {
		return nil, err
	}
	result, err := unmarshalToRecordUnion(response)
	if err != nil {
		return nil, err
	}
	// after request interceptor
	return resourceCaller.doAfterRequest(ctx, result)
}

// doRequestWithRetries attempts to perform an HTTP request using doRequest,
// retrying up to 3 times if the request fails with a 403 Forbidden API error.
// It uses the provided context for cancellation support. If a non-retryable
// error occurs, it returns immediately without retrying.
func doRequestWithRetries(ctx context.Context, s *VMSSession, verb, url string, body Params, headers []http.Header) (Renderable, error) {
	var (
		err    error
		result Renderable
	)
	for i := 0; i < maxRetries; i++ {
		result, err = doRequest(ctx, s, verb, url, body, headers)
		if err != nil && IsApiError(err) {
			statusCode := err.(*ApiError).StatusCode
			if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
				if authErr := s.auth.authorize(); authErr != nil {
					return nil, authErr
				}
				continue
			}
		}
		break
	}
	return result, err
}
