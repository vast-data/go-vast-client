package vast_client

import (
	"bytes"
	"crypto/tls"
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

// createAuthenticator returns a singleton Authenticator instance based on the provided VMSConfig.
// It ensures only one instance per authenticator type exists.
func createAuthenticator(config *VMSConfig) (Authenticator, error) {
	var authenticator Authenticator
	if config.Username != "" && config.Password != "" {
		authenticator = &JWTAuthenticator{
			Host:      config.Host,
			Port:      config.Port,
			SslVerify: config.SslVerify,
			Username:  config.Username,
			Password:  config.Password,
			Token:     &jwtToken{},
		}
	}
	if config.ApiToken != "" {
		authenticator = &ApiRTokenAuthenticator{
			Host:      config.Host,
			Port:      config.Port,
			SslVerify: config.SslVerify,
			Token:     config.ApiToken,
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
	var resp *http.Response
	path := url.URL{
		Scheme: "https",
		Host:   auth.Host,
		Path:   "api/token/refresh/",
	}
	body, err := json.Marshal(map[string]string{"refresh": auth.Token.Refresh})
	if err != nil {
		return nil, err
	}
	resp, err = client.Post(path.String(), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	return resp, nil
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
	return client.Post(path.String(), "application/json", bytes.NewBuffer(body))
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
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Prevent following redirects (like 301, 302)
			return http.ErrUseLastResponse
		},
	}

	if auth.initialized {
		resp, err = auth.refreshToken(client)
	} else {
		resp, err = auth.acquireToken(client)
		auth.setInitialized(true)
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
	headers.Add("Authorization", "Bearer "+auth.Token.Access)
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
}

func (auth *ApiRTokenAuthenticator) authorize() error {
	// No-op for ApiRTokenAuthenticator
	return nil
}

func (auth *ApiRTokenAuthenticator) setAuthHeader(headers *http.Header) {
	headers.Add("Authorization", "Api-Token "+auth.Token)
}

func (auth *ApiRTokenAuthenticator) equal(other Authenticator) bool {
	otherAuth, ok := other.(*ApiRTokenAuthenticator)
	if !ok {
		return false
	}
	return auth.Token == otherAuth.Token &&
		auth.Host == otherAuth.Host &&
		auth.Port == otherAuth.Port &&
		auth.SslVerify == otherAuth.SslVerify
}

func (auth *ApiRTokenAuthenticator) setInitialized(_ bool) {
	// No-op
}
