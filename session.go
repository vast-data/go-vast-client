package vast_client

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
	maxRetries            = 3
)

type RESTSession interface {
	Get(context.Context, string, Params) (Renderable, error)
	Post(context.Context, string, Params) (Renderable, error)
	Put(context.Context, string, Params) (Renderable, error)
	Patch(context.Context, string, Params) (Renderable, error)
	Delete(context.Context, string, Params) (Renderable, error)
	GetConfig() *VMSConfig
	GetAuthenticator() Authenticator
}

// ApiError represents an error returned from an API request.
type ApiError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

// Error implements the error interface.
func (e *ApiError) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("response body: %s", e.Body)
	}
	return fmt.Sprintf(
		"%s request to %s returned status code %d"+
			" — response body: %s", e.Method, e.URL, e.StatusCode, e.Body,
	)
}

func IsApiError(err error) bool {
	var apiErr *ApiError
	if errors.As(err, &apiErr) {
		return true
	}
	return false
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

type VMSSession struct {
	config *VMSConfig
	client *http.Client
	auth   Authenticator
}

type VMSSessionMethod func(context.Context, string, Params) (Renderable, error)

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

func request[T RecordUnion](
	ctx context.Context,
	r InterceptableVastResourceAPI,
	verb, path, apiVer string,
	params, body Params,
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
	url, err := buildUrl(session, path, query, apiVer)
	if err != nil {
		return nil, err
	}

	response, err := vmsMethod(ctx, url, body)
	if err != nil {
		return nil, err
	}

	if typeMatch[Record](response) {
		// Some resources return single record
		// although query typically return list for others (for instance NonLocalUser)
		// We want to eliminate this discrepancy by casting Record to RecordSet
		var zero T
		if typeMatch[RecordSet](Renderable(zero)) {
			if !response.(Record).empty() {
				response = RecordSet{response.(Record)}
			} else {
				response = RecordSet{}
			}
		}
	}
	resultVal, ok := response.(T)
	if !ok {
		return nil, fmt.Errorf(
			"unexpected response type for request to %s: got %T, expected %T — consider converting the response to the expected type inside the doAfterRequest interceptor",
			url,
			response,
			*new(T),
		)
	}
	return resultVal, nil
}

func (s *VMSSession) Get(ctx context.Context, url string, _ Params) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodGet, url, nil, nil)
}

func (s *VMSSession) Post(ctx context.Context, url string, body Params) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodPost, url, body, nil)
}

func (s *VMSSession) Put(ctx context.Context, url string, body Params) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodPut, url, body, nil)
}

func (s *VMSSession) Patch(ctx context.Context, url string, body Params) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodPatch, url, body, nil)
}

func (s *VMSSession) Delete(ctx context.Context, url string, body Params) (Renderable, error) {
	return doRequestWithRetries(ctx, s, http.MethodDelete, url, body, nil)
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
		"Authorization": []string{"Basic " + encoded},
		"Accept":        []string{"application/openapi+json"},
	}}
	return doRequest(ctx, s, http.MethodGet, url+"?format=openapi", nil, headers)
}

func (s *VMSSession) GetConfig() *VMSConfig {
	return s.config
}
func (s *VMSSession) GetAuthenticator() Authenticator {
	return s.auth
}

func setupHeaders(s RESTSession, r *http.Request) error {
	s.GetAuthenticator().setAuthHeader(&r.Header)
	r.Header.Add("Accept", ApplicationJson)
	r.Header.Add("Content-type", ApplicationJson)
	r.Header.Set("User-Agent", s.GetConfig().UserAgent)
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
		resourceCaller = dummyResource
	} else {
		resourceCaller = originResource
	}
	// Check if called resource can be used with current version of VAST cluster.
	if err = checkVastResourceVersionCompat(ctx, resourceCaller); err != nil {
		return nil, err
	}
	// Convert to full URI if needed.
	if url, err = pathToUrl(s, url); err != nil {
		return nil, err
	}
	if body == nil {
		requestData = bytes.NewReader(nil)
	} else {
		if requestData, err = body.ToBody(); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, verb, url, requestData)
	if beforeRequestData, err = body.ToBody(); err != nil {
		return nil, err
	}
	if headers != nil {
		// Setup custom headers if provided
		for _, header := range headers {
			for key, values := range header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}
		}
	} else {
		// Setup default headers
		if err = setupHeaders(s, req); err != nil {
			return nil, err
		}
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
				if statusCode == http.StatusUnauthorized {
					// Probably refresh token is expired. Need full re-authentication
					s.auth.setInitialized(false)
				}
				if err = s.auth.authorize(); err != nil {
					return nil, err
				}
				continue
			}
		}
		break
	}
	return result, err
}
