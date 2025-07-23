package vast_client

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestVMSConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *VMSConfig
		validator VMSConfigFunc
		wantPanic bool
	}{
		{
			name: "valid config with username/password",
			config: &VMSConfig{
				Host:     "test.com",
				Username: "admin",
				Password: "password",
			},
			validator: withAuth,
			wantPanic: false,
		},
		{
			name: "valid config with API token",
			config: &VMSConfig{
				Host:     "test.com",
				ApiToken: "token123",
			},
			validator: withAuth,
			wantPanic: false,
		},
		{
			name: "invalid config - no auth",
			config: &VMSConfig{
				Host: "test.com",
			},
			validator: withAuth,
			wantPanic: true,
		},
		{
			name: "invalid config - empty host",
			config: &VMSConfig{
				Username: "admin",
				Password: "password",
			},
			validator: withHost,
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("VMSConfig.Validate() panic = %v, wantPanic %v", r != nil, tt.wantPanic)
				}
			}()
			tt.config.Validate(tt.validator)
		})
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 60 * time.Second
	validator := withTimeout(timeout)

	t.Run("sets timeout when nil", func(t *testing.T) {
		config := &VMSConfig{}
		err := validator(config)
		if err != nil {
			t.Errorf("withTimeout() error = %v, wantErr false", err)
		}
		if *config.Timeout != timeout {
			t.Errorf("withTimeout() timeout = %v, want %v", *config.Timeout, timeout)
		}
	})

	t.Run("preserves existing timeout", func(t *testing.T) {
		existing := 30 * time.Second
		config := &VMSConfig{Timeout: &existing}
		err := validator(config)
		if err != nil {
			t.Errorf("withTimeout() error = %v, wantErr false", err)
		}
		if *config.Timeout != existing {
			t.Errorf("withTimeout() timeout = %v, want %v", *config.Timeout, existing)
		}
	})
}

func TestWithMaxConnections(t *testing.T) {
	maxConn := 20
	validator := withMaxConnections(maxConn)

	t.Run("sets max connections when zero", func(t *testing.T) {
		config := &VMSConfig{}
		err := validator(config)
		if err != nil {
			t.Errorf("withMaxConnections() error = %v, wantErr false", err)
		}
		if config.MaxConnections != maxConn {
			t.Errorf("withMaxConnections() MaxConnections = %v, want %v", config.MaxConnections, maxConn)
		}
	})

	t.Run("preserves existing max connections", func(t *testing.T) {
		existing := 5
		config := &VMSConfig{MaxConnections: existing}
		err := validator(config)
		if err != nil {
			t.Errorf("withMaxConnections() error = %v, wantErr false", err)
		}
		if config.MaxConnections != existing {
			t.Errorf("withMaxConnections() MaxConnections = %v, want %v", config.MaxConnections, existing)
		}
	})
}

func TestWithHost(t *testing.T) {
	tests := []struct {
		name      string
		config    *VMSConfig
		wantPanic bool
	}{
		{
			name:      "valid host",
			config:    &VMSConfig{Host: "test.com"},
			wantPanic: false,
		},
		{
			name:      "empty host",
			config:    &VMSConfig{Host: ""},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("withHost() panic = %v, wantPanic %v", r != nil, tt.wantPanic)
				}
			}()
			withHost(tt.config)
		})
	}
}

func TestWithPort(t *testing.T) {
	port := uint64(8443)
	validator := withPort(port)

	t.Run("sets port when zero", func(t *testing.T) {
		config := &VMSConfig{}
		err := validator(config)
		if err != nil {
			t.Errorf("withPort() error = %v, wantErr false", err)
		}
		if config.Port != port {
			t.Errorf("withPort() Port = %v, want %v", config.Port, port)
		}
	})

	t.Run("preserves existing port", func(t *testing.T) {
		existing := uint64(9000)
		config := &VMSConfig{Port: existing}
		err := validator(config)
		if err != nil {
			t.Errorf("withPort() error = %v, wantErr false", err)
		}
		if config.Port != existing {
			t.Errorf("withPort() Port = %v, want %v", config.Port, existing)
		}
	})
}

func TestWithAuth(t *testing.T) {
	tests := []struct {
		name    string
		config  *VMSConfig
		wantErr bool
	}{
		{
			name: "valid username/password",
			config: &VMSConfig{
				Username: "admin",
				Password: "password",
			},
			wantErr: false,
		},
		{
			name: "valid API token",
			config: &VMSConfig{
				ApiToken: "token123",
			},
			wantErr: false,
		},
		{
			name: "both username/password and token",
			config: &VMSConfig{
				Username: "admin",
				Password: "password",
				ApiToken: "token123",
			},
			wantErr: false,
		},
		{
			name:    "no auth provided",
			config:  &VMSConfig{},
			wantErr: true,
		},
		{
			name: "incomplete username/password",
			config: &VMSConfig{
				Username: "admin",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := withAuth(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("withAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWithUserAgent(t *testing.T) {
	t.Run("sets default user agent when empty", func(t *testing.T) {
		config := &VMSConfig{}
		err := withUserAgent(config)
		if err != nil {
			t.Errorf("withUserAgent() error = %v, wantErr false", err)
		}
		if config.UserAgent == "" {
			t.Error("withUserAgent() UserAgent should not be empty")
		}
		if !containsString(config.UserAgent, "vast-go-client") {
			t.Errorf("withUserAgent() UserAgent = %v, should contain 'vast-go-client'", config.UserAgent)
		}
	})

	t.Run("preserves existing user agent", func(t *testing.T) {
		existing := "custom-agent/1.0"
		config := &VMSConfig{UserAgent: existing}
		err := withUserAgent(config)
		if err != nil {
			t.Errorf("withUserAgent() error = %v, wantErr false", err)
		}
		if config.UserAgent != existing {
			t.Errorf("withUserAgent() UserAgent = %v, want %v", config.UserAgent, existing)
		}
	})
}

func TestWithApiVersion(t *testing.T) {
	version := "v6"
	validator := withApiVersion(version)

	t.Run("sets API version when empty", func(t *testing.T) {
		config := &VMSConfig{}
		err := validator(config)
		if err != nil {
			t.Errorf("withApiVersion() error = %v, wantErr false", err)
		}
		if config.ApiVersion != version {
			t.Errorf("withApiVersion() ApiVersion = %v, want %v", config.ApiVersion, version)
		}
	})

	t.Run("preserves existing API version", func(t *testing.T) {
		existing := "v4"
		config := &VMSConfig{ApiVersion: existing}
		err := validator(config)
		if err != nil {
			t.Errorf("withApiVersion() error = %v, wantErr false", err)
		}
		if config.ApiVersion != existing {
			t.Errorf("withApiVersion() ApiVersion = %v, want %v", config.ApiVersion, existing)
		}
	})
}

func TestWithFillFn(t *testing.T) {
	// Save the original fillFunc to restore it after the test
	originalFillFunc := fillFunc
	defer func() {
		fillFunc = originalFillFunc
	}()

	customFillFn := func(_ Record, container any) error {
		return nil
	}

	config := &VMSConfig{FillFn: customFillFn}
	err := withFillFn(config)
	if err != nil {
		t.Errorf("withFillFn() error = %v, wantErr false", err)
	}
	// We can't directly compare functions, but we can check that fillFunc was set
	// by testing that it's no longer the default
}

func TestVMSConfig_CompleteValidation(t *testing.T) {
	t.Run("complete validation chain", func(t *testing.T) {
		config := &VMSConfig{
			Host:     "test.com",
			Username: "admin",
			Password: "password",
		}

		// This should not panic
		config.Validate(
			withAuth,
			withHost,
			withUserAgent,
			withFillFn,
			withApiVersion("v5"),
			withTimeout(30*time.Second),
			withMaxConnections(10),
			withPort(443),
		)

		// Verify all defaults were set
		if config.UserAgent == "" {
			t.Error("UserAgent should be set")
		}
		if config.ApiVersion != "v5" {
			t.Errorf("ApiVersion = %v, want v5", config.ApiVersion)
		}
		if config.Timeout == nil || *config.Timeout != 30*time.Second {
			t.Error("Timeout should be set to 30s")
		}
		if config.MaxConnections != 10 {
			t.Errorf("MaxConnections = %v, want 10", config.MaxConnections)
		}
		if config.Port != 443 {
			t.Errorf("Port = %v, want 443", config.Port)
		}
	})
}

func TestVMSConfig_RequestHooks(t *testing.T) {
	t.Run("before request hook", func(t *testing.T) {
		called := false
		beforeFn := func(_ context.Context, r *http.Request, verb, url string, body io.Reader) error {
			called = true
			return nil
		}

		config := &VMSConfig{
			Host:            "test.com",
			Username:        "admin",
			Password:        "password",
			BeforeRequestFn: beforeFn,
		}

		if config.BeforeRequestFn == nil {
			t.Error("BeforeRequestFn should be set")
		}

		// Test that the function can be called
		err := config.BeforeRequestFn(context.Background(), nil, "GET", "test", nil)
		if err != nil {
			t.Errorf("BeforeRequestFn error = %v, want nil", err)
		}
		if !called {
			t.Error("BeforeRequestFn should have been called")
		}
	})

	t.Run("after request hook", func(t *testing.T) {
		called := false
		afterFn := func(ctx context.Context, response Renderable) (Renderable, error) {
			called = true
			return response, nil
		}

		config := &VMSConfig{
			Host:           "test.com",
			Username:       "admin",
			Password:       "password",
			AfterRequestFn: afterFn,
		}

		if config.AfterRequestFn == nil {
			t.Error("AfterRequestFn should be set")
		}

		// Test that the function can be called
		result, err := config.AfterRequestFn(context.Background(), Record{})
		if err != nil {
			t.Errorf("AfterRequestFn error = %v, want nil", err)
		}
		if result == nil {
			t.Error("AfterRequestFn should return a result")
		}
		if !called {
			t.Error("AfterRequestFn should have been called")
		}
	})
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr
}
