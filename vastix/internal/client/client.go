package client

import (
	"crypto/sha256"
	"fmt"
	"sync"

	vastclient "github.com/vast-data/go-vast-client"
)

// RestClientConfig holds configuration for a REST client
type RestClientConfig struct {
	Host       string
	Port       int64
	Username   string
	Password   string
	ApiToken   string
	Tenant     string
	SslVerify  bool
	ApiVersion string
}

// Service manages REST clients with caching based on configuration hash
type Service struct {
	clients map[string]*vastclient.VMSRest
	mu      sync.RWMutex
}

var (
	clientInstance *Service
	clientOnce     sync.Once
)

// NewRestService creates or returns the singleton REST client service
func NewRestService() *Service {
	clientOnce.Do(func() {
		clientInstance = &Service{
			clients: make(map[string]*vastclient.VMSRest),
		}
	})
	return clientInstance
}

// GetGlobalRestService returns the global REST client service instance
func GetGlobalRestService() *Service {
	return NewRestService()
}

// generateConfigHash creates a unique hash for the given configuration
func (s *Service) generateConfigHash(config RestClientConfig) string {
	configStr := fmt.Sprintf("%s:%d:%s:%s:%s:%s:%t:%s",
		config.Host, config.Port, config.Username, config.Password,
		config.ApiToken, config.Tenant, config.SslVerify, config.ApiVersion)

	hash := sha256.Sum256([]byte(configStr))
	return fmt.Sprintf("%x", hash)
}

// GetOrCreateClient returns an existing client for the configuration or creates a new one
func (s *Service) GetOrCreateClient(config RestClientConfig) (*vastclient.VMSRest, error) {
	hash := s.generateConfigHash(config)

	s.mu.RLock()
	if client, exists := s.clients[hash]; exists {
		s.mu.RUnlock()
		return client, nil
	}
	s.mu.RUnlock()

	// Need to create new client
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check in case another goroutine created it
	if client, exists := s.clients[hash]; exists {
		return client, nil
	}

	// Create new client
	client, err := NewRest(
		config.Host, config.Port, config.Username, config.Password,
		config.ApiToken, config.Tenant, config.SslVerify, config.ApiVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	s.clients[hash] = client
	return client, nil
}

// RemoveClient removes a client with the given configuration
func (s *Service) RemoveClient(config RestClientConfig) {
	hash := s.generateConfigHash(config)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, hash)
}

// ClearAllClients removes all cached clients
func (s *Service) ClearAllClients() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients = make(map[string]*vastclient.VMSRest)
}

// GetClientCount returns the number of cached clients
func (s *Service) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.clients)
}

func NewRest(
	host string,
	port int64,
	username, password, apiToken, tenant string,
	sslVerify bool,
	apiVersion string,
) (*vastclient.VMSRest, error) {
	vmsConfig := &vastclient.VMSConfig{
		Host:       host,
		Port:       uint64(port),
		Username:   username,
		Password:   password,
		ApiToken:   apiToken,
		SslVerify:  sslVerify,
		Tenant:     tenant,
		UserAgent:  getUserAgent(),
		ApiVersion: apiVersion,

		BeforeRequestFn: BeforeRequestFnCallback,
		AfterRequestFn:  AfterRequestFnCallback,
	}

	return vastclient.NewVMSRest(vmsConfig)
}

// Global convenience functions for easier access
var globalRestService *Service

// InitGlobalRestService initializes the global REST service instance
func InitGlobalRestService() *Service {
	globalRestService = NewRestService()
	return globalRestService
}

// GetGlobalClient returns a client for the given configuration using the global service
func GetGlobalClient(config RestClientConfig) (*vastclient.VMSRest, error) {
	if globalRestService == nil {
		globalRestService = InitGlobalRestService()
	}
	return globalRestService.GetOrCreateClient(config)
}

// ClearGlobalClients clears all cached clients in the global service
func ClearGlobalClients() {
	if globalRestService != nil {
		globalRestService.ClearAllClients()
	}
}
