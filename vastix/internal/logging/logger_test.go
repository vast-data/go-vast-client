package logging

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestLoggingService(t *testing.T) {
	// Test creating a new logging service
	service, closeFn, err := New()
	if err != nil {
		t.Fatalf("Failed to create logging service: %v", err)
	}
	defer func() {
		if closeFn != nil {
			closeFn()
		}
	}()

	if service == nil {
		t.Error("Expected non-nil service")
	}

	logger := service.GetLogger()
	if logger == nil {
		t.Error("Expected non-nil logger")
	}

	// Test that we can log without errors
	logger.Info("test message")
}

func TestGetGlobalLogger(t *testing.T) {
	// Set log level for consistent testing
	os.Setenv("LOG_LEVEL", "INFO")
	defer os.Unsetenv("LOG_LEVEL")

	// Initialize global logger first
	closeFn, err := InitGlobalLogger()
	if err != nil {
		t.Fatalf("Failed to init global logger: %v", err)
	}
	defer func() {
		if closeFn != nil {
			closeFn()
		}
	}()

	logger1 := GetGlobalLogger()
	logger2 := GetGlobalLogger()

	if logger1 == nil {
		t.Error("Expected non-nil logger from first call")
	}

	if logger2 == nil {
		t.Error("Expected non-nil logger from second call")
	}

	// Both calls should return the same instance (singleton pattern)
	if logger1 != logger2 {
		t.Error("Expected same logger instance from multiple calls")
	}
}

func TestGetAuxLogger(t *testing.T) {
	logger1 := GetAuxLogger()
	logger2 := GetAuxLogger()

	if logger1 == nil {
		t.Error("Expected non-nil aux logger from first call")
	}

	if logger2 == nil {
		t.Error("Expected non-nil aux logger from second call")
	}

	// Both calls should return the same instance (singleton pattern)
	if logger1 != logger2 {
		t.Error("Expected same aux logger instance from multiple calls")
	}
}

func TestLogLevelEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
	}{
		{"debug level", "DEBUG"},
		{"info level", "INFO"},
		{"warn level", "WARN"},
		{"error level", "ERROR"},
		{"invalid level defaults to info", "INVALID"},
		{"empty level defaults to info", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.logLevel != "" {
				os.Setenv("LOG_LEVEL", tt.logLevel)
			} else {
				os.Unsetenv("LOG_LEVEL")
			}
			defer os.Unsetenv("LOG_LEVEL")

			// Create new service with this log level
			service, closeFn, err := New()
			if err != nil {
				t.Fatalf("Failed to create logging service: %v", err)
			}
			defer func() {
				if closeFn != nil {
					closeFn()
				}
			}()

			logger := service.GetLogger()
			if logger == nil {
				t.Error("Expected non-nil logger")
			}

			// Test that we can log without errors
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")
		})
	}
}

func TestGlobalLoggerFunctions(t *testing.T) {
	// Initialize global logger
	closeFn, err := InitGlobalLogger()
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}
	defer func() {
		if closeFn != nil {
			closeFn()
		}
	}()

	// Test global logger functions
	logger := GetGlobalLogger()
	if logger == nil {
		t.Error("Expected non-nil global logger")
	}

	// Test convenience functions
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	// Test with fields
	Info("structured log", zap.String("key", "value"), zap.Int("number", 42))
}

func TestAuxLogger(t *testing.T) {
	// Create service to initialize aux logger
	_, closeFn, err := New()
	if err != nil {
		t.Fatalf("Failed to create logging service: %v", err)
	}
	defer func() {
		if closeFn != nil {
			closeFn()
		}
	}()

	auxLogger := GetAuxLogger()
	if auxLogger == nil {
		t.Error("Expected non-nil aux logger")
	}

	// Test aux logger functions
	AuxLog("aux log message")
	AuxLogf("aux log formatted: %s", "test")

	// Direct use of aux logger
	auxLogger.Println("direct aux logger message")
}

func TestGetVastixDir(t *testing.T) {
	dir, err := GetVastixDir()
	if err != nil {
		t.Fatalf("Failed to get vastix dir: %v", err)
	}

	if dir == "" {
		t.Error("Expected non-empty directory path")
	}

	// Check that directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Vastix directory was not created")
	}
}

// Benchmark tests for logger performance
func BenchmarkLoggingService_Create(b *testing.B) {
	for i := 0; i < b.N; i++ {
		service, closeFn, err := New()
		if err != nil {
			b.Fatalf("Failed to create service: %v", err)
		}
		if closeFn != nil {
			closeFn()
		}
		_ = service
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	service, closeFn, err := New()
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}
	defer func() {
		if closeFn != nil {
			closeFn()
		}
	}()

	logger := service.GetLogger()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}
