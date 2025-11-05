package core

import (
	"testing"
	"time"
)

func TestVMSConfig_Validate(t *testing.T) {
	t.Run("valid config with all validators", func(t *testing.T) {
		config := &VMSConfig{
			Host:     "localhost",
			Port:     443,
			Username: "admin",
			Password: "password",
		}
		// Should not panic
		config.Validate(WithHost, WithAuth)
	})

	t.Run("missing host panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for missing host")
			}
		}()
		config := &VMSConfig{
			Port:     443,
			Username: "admin",
			Password: "password",
		}
		config.Validate(WithHost)
	})

	t.Run("missing auth returns error", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for missing auth")
			}
		}()
		config := &VMSConfig{
			Host: "localhost",
			Port: 443,
		}
		config.Validate(WithAuth)
	})
}

func TestWithTimeout(t *testing.T) {
	config := &VMSConfig{}
	timeout := 30 * time.Second

	fn := WithTimeout(timeout)
	err := fn(config)

	if err != nil {
		t.Errorf("WithTimeout() error = %v", err)
	}
	if config.Timeout == nil {
		t.Error("WithTimeout() did not set timeout")
	} else if *config.Timeout != timeout {
		t.Errorf("WithTimeout() timeout = %v, want %v", *config.Timeout, timeout)
	}
}

func TestWithMaxConnections(t *testing.T) {
	config := &VMSConfig{}
	maxConns := 100

	fn := WithMaxConnections(maxConns)
	err := fn(config)

	if err != nil {
		t.Errorf("WithMaxConnections() error = %v", err)
	}
	if config.MaxConnections != maxConns {
		t.Errorf("WithMaxConnections() MaxConnections = %d, want %d", config.MaxConnections, maxConns)
	}
}

func TestWithHost(t *testing.T) {
	t.Run("valid host", func(t *testing.T) {
		config := &VMSConfig{Host: "localhost"}
		err := WithHost(config)
		if err != nil {
			t.Errorf("WithHost() error = %v", err)
		}
	})

	t.Run("empty host panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for empty host")
			}
		}()
		config := &VMSConfig{Host: ""}
		_ = WithHost(config)
	})
}

func TestWithPort(t *testing.T) {
	config := &VMSConfig{}
	port := uint64(8443)

	fn := WithPort(port)
	err := fn(config)

	if err != nil {
		t.Errorf("WithPort() error = %v", err)
	}
	if config.Port != port {
		t.Errorf("WithPort() Port = %d, want %d", config.Port, port)
	}
}

func TestWithAuth(t *testing.T) {
	t.Run("valid username/password", func(t *testing.T) {
		config := &VMSConfig{
			Username: "admin",
			Password: "password",
		}
		err := WithAuth(config)
		if err != nil {
			t.Errorf("WithAuth() error = %v", err)
		}
	})

	t.Run("valid api token", func(t *testing.T) {
		config := &VMSConfig{
			ApiToken: "token123",
		}
		err := WithAuth(config)
		if err != nil {
			t.Errorf("WithAuth() error = %v", err)
		}
	})

	t.Run("missing auth", func(t *testing.T) {
		config := &VMSConfig{}
		err := WithAuth(config)
		if err == nil {
			t.Error("WithAuth() expected error for missing auth")
		}
	})
}

func TestWithUserAgent(t *testing.T) {
	config := &VMSConfig{}
	err := WithUserAgent(config)

	if err != nil {
		t.Errorf("WithUserAgent() error = %v", err)
	}
	if config.UserAgent == "" {
		t.Error("WithUserAgent() did not set UserAgent")
	}
}

func TestWithApiVersion(t *testing.T) {
	config := &VMSConfig{}
	apiVersion := "v2"

	fn := WithApiVersion(apiVersion)
	err := fn(config)

	if err != nil {
		t.Errorf("WithApiVersion() error = %v", err)
	}
	if config.ApiVersion != apiVersion {
		t.Errorf("WithApiVersion() ApiVersion = %s, want %s", config.ApiVersion, apiVersion)
	}
}

func TestWithFillFn(t *testing.T) {
	config := &VMSConfig{
		FillFn: func(r Record, container any) error {
			return nil
		},
	}

	err := WithFillFn(config)

	if err != nil {
		t.Errorf("WithFillFn() error = %v", err)
	}
	// Verify fillFunc was set globally
	if fillFunc == nil {
		t.Error("WithFillFn() did not set global fillFunc")
	}
}
