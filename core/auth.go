package core

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var authenticators []Authenticator

type Authenticator interface {
	authorize() error
	setAuthHeader(headers *http.Header)
	equal(other Authenticator) bool
	setInitialized(bool)
}

// createAuthenticator creates a new Authenticator instance based on the provided VMSConfig.
// Each session gets its own authenticator instance to avoid global state issues.
func createAuthenticator(config *VMSConfig) (Authenticator, error) {
	var authenticator Authenticator

	// Priority: ApiToken > BasicAuth > JWT
	if config.ApiToken != "" {
		authenticator = &ApiRTokenAuthenticator{
			Host:      config.Host,
			Port:      config.Port,
			SslVerify: config.SslVerify,
			Token:     config.ApiToken,
			Tenant:    config.Tenant,
		}
	} else if config.UseBasicAuth && config.Username != "" && config.Password != "" {
		authenticator = &BaseAuthAuthenticator{
			Host:      config.Host,
			Port:      config.Port,
			SslVerify: config.SslVerify,
			Username:  config.Username,
			Password:  config.Password,
			Tenant:    config.Tenant,
		}
	} else if config.Username != "" && config.Password != "" {
		authenticator = &JWTAuthenticator{
			Host:      config.Host,
			Port:      config.Port,
			SslVerify: config.SslVerify,
			Username:  config.Username,
			Password:  config.Password,
			Tenant:    config.Tenant,
			Token:     &jwtToken{},
		}
	}
	if authenticator != nil {
		for _, existingAuthenticator := range authenticators {
			if existingAuthenticator.equal(authenticator) {
				return existingAuthenticator, nil
			}
		}
		if err := authenticator.authorize(); err != nil {
			return nil, err
		}
		authenticators = append(authenticators, authenticator)
		return authenticator, nil
	}

	panic("CreateAuthenticator: neither username/password nor apiToken are provided")
}

type jwtToken struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type JWTAuthenticator struct {
	Host        string
	Port        uint64
	SslVerify   bool
	Username    string
	Password    string
	Token       *jwtToken
	Tenant      string
	initialized bool
}

func parseToken(rsp *http.Response) (*jwtToken, error) {
	var tokens jwtToken
	out, e := io.ReadAll(rsp.Body)
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal(out, &tokens)
	if e != nil {
		return nil, e
	}
	return &tokens, nil
}

func (auth *JWTAuthenticator) refreshToken(client *http.Client) (*http.Response, error) {
	path := url.URL{
		Scheme: "https",
		Host:   auth.Host,
		Path:   "api/token/refresh/",
	}
	body, err := json.Marshal(map[string]string{"refresh": auth.Token.Refresh})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, path.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderContentType, ContentTypeJSON)
	if auth.Tenant != "" {
		req.Header.Set(HeaderXTenantName, auth.Tenant)
	}

	return client.Do(req)
}

func (auth *JWTAuthenticator) acquireToken(client *http.Client) (*http.Response, error) {
	// obtain new access & refresh tokens
	userPass := map[string]string{"username": auth.Username, "password": auth.Password}
	server := auth.Host + ":" + strconv.FormatUint(auth.Port, 10)
	body, err := json.Marshal(userPass)
	if err != nil {
		return nil, err
	}
	// Generate URL to obtain token keys
	path := url.URL{
		Scheme: "https",
		Host:   server,
		Path:   "api/token/",
	}
	req, err := http.NewRequest(http.MethodPost, path.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderContentType, ContentTypeJSON)
	if auth.Tenant != "" {
		req.Header.Set(HeaderXTenantName, auth.Tenant)
	}

	return client.Do(req)
}

func (auth *JWTAuthenticator) authorize() error {
	var (
		resp *http.Response
		err  error
	)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !auth.SslVerify},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   20 * time.Second,
	}
	if auth.initialized {
		resp, err = auth.refreshToken(client)
		// If there is an error while getting new token using refresh token and
		// that error is API error with status code 401, then refresh token is also
		// expired. Need to re-authenticate.
		if err != nil && IsApiError(err) {
			statusCode := err.(*ApiError).StatusCode
			if statusCode == http.StatusUnauthorized {
				resp, err = auth.acquireToken(client)
				auth.setInitialized(true)
			}
		}
	} else {
		resp, err = auth.acquireToken(client)
		auth.setInitialized(true)
	}
	if err != nil {
		return err
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	if err = validateResponse(resp, auth.Host, auth.Port); err != nil {
		return err
	}
	// Read response
	token, err := parseToken(resp)
	if err != nil {
		return err
	}
	auth.Token = token
	return nil
}

func (auth *JWTAuthenticator) setAuthHeader(headers *http.Header) {
	headers.Add(HeaderAuthorization, AuthTypeBearer+" "+auth.Token.Access)
	if auth.Tenant != "" {
		headers.Add(HeaderXTenantName, auth.Tenant)
	}
}

func (auth *JWTAuthenticator) equal(other Authenticator) bool {
	otherAuth, ok := other.(*JWTAuthenticator)
	if !ok {
		return false
	}
	return auth.Username == otherAuth.Username &&
		auth.Password == otherAuth.Password &&
		auth.Host == otherAuth.Host &&
		auth.Port == otherAuth.Port &&
		auth.Tenant == otherAuth.Tenant &&
		auth.SslVerify == otherAuth.SslVerify
}

func (auth *JWTAuthenticator) setInitialized(state bool) {
	auth.initialized = state
}

type ApiRTokenAuthenticator struct {
	Host      string
	Port      uint64
	SslVerify bool
	Token     string
	Tenant    string
}

func (auth *ApiRTokenAuthenticator) authorize() error {
	// No-op for ApiRTokenAuthenticator
	return nil
}

func (auth *ApiRTokenAuthenticator) setAuthHeader(headers *http.Header) {
	headers.Add("Authorization", "Api-Token "+auth.Token)
	if auth.Tenant != "" {
		headers.Add(HeaderXTenantName, auth.Tenant)
	}
}

func (auth *ApiRTokenAuthenticator) equal(other Authenticator) bool {
	otherAuth, ok := other.(*ApiRTokenAuthenticator)
	if !ok {
		return false
	}
	return auth.Token == otherAuth.Token &&
		auth.Host == otherAuth.Host &&
		auth.Port == otherAuth.Port &&
		auth.Tenant == otherAuth.Tenant &&
		auth.SslVerify == otherAuth.SslVerify
}

func (auth *ApiRTokenAuthenticator) setInitialized(_ bool) {
	// No-op
}

type BaseAuthAuthenticator struct {
	Host        string
	Port        uint64
	SslVerify   bool
	Username    string
	Password    string
	Tenant      string
	encodedAuth string // Cached Base64-encoded credentials
}

func (auth *BaseAuthAuthenticator) authorize() error {
	// Pre-compute and cache the Base64-encoded Basic Auth credentials
	// This is called once during setup, avoiding repeated encoding on each request
	authStr := auth.Username + ":" + auth.Password
	auth.encodedAuth = base64.StdEncoding.EncodeToString([]byte(authStr))
	return nil
}

func (auth *BaseAuthAuthenticator) setAuthHeader(headers *http.Header) {
	// Use the pre-encoded credentials from authorize()
	headers.Add(HeaderAuthorization, AuthTypeBasic+" "+auth.encodedAuth)
	if auth.Tenant != "" {
		headers.Add(HeaderXTenantName, auth.Tenant)
	}
}

func (auth *BaseAuthAuthenticator) equal(other Authenticator) bool {
	otherAuth, ok := other.(*BaseAuthAuthenticator)
	if !ok {
		return false
	}
	return auth.Username == otherAuth.Username &&
		auth.Password == otherAuth.Password &&
		auth.Host == otherAuth.Host &&
		auth.Port == otherAuth.Port &&
		auth.Tenant == otherAuth.Tenant &&
		auth.SslVerify == otherAuth.SslVerify
}

func (auth *BaseAuthAuthenticator) setInitialized(_ bool) {
	// No-op for Basic Auth
}
