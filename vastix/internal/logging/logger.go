package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Service struct {
	logger *zap.Logger
}

var (
	logInstance *Service
	auxLogger   *log.Logger
)

// GetVastixDir returns the .vastix directory path, creating it if it doesn't exist
func GetVastixDir() (string, error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	// Create .vastix directory in user home
	vastixDir := filepath.Join(homeDir, ".vastix")
	if err := os.MkdirAll(vastixDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .vastix directory: %v", err)
	}

	return vastixDir, nil
}

// New creates a new logging service instance (singleton)
func New() (*Service, func(), error) {
	// Reuse existing instance
	if logInstance != nil {
		return logInstance, nil, nil
	}

	vastixDir, err := GetVastixDir()
	if err != nil {
		return nil, nil, err
	}

	// Create logs directory within .vastix
	logsDir := filepath.Join(vastixDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create logs directory: %v", err)
	}

	// Log file path
	logPath := filepath.Join(logsDir, "app.log")
	auxLogPath := filepath.Join(logsDir, "aux.log")

	// Remove existing aux log file to recreate it on each app restart
	if err := os.Remove(auxLogPath); err != nil && !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("failed to remove existing aux log file: %v", err)
	}

	auxLogFile, err := tea.LogToFile(auxLogPath, "aux")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create aux log file: %v", err)
	}

	// Initialize auxiliary logger singleton if not already created
	if auxLogger == nil {
		auxLogger = log.New(auxLogFile, "", log.LstdFlags)
	}

	// Setup logger with file rotation (file-only, no console)
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    2, // megabytes
		MaxBackups: 5,
		MaxAge:     15, // days
		Compress:   true,
	})

	// Configure JSON encoder
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	encoder := zapcore.NewJSONEncoder(cfg.EncoderConfig)

	// Use INFO level for production logging, but allow override via environment variable
	level := zapcore.InfoLevel
	if logLevelEnv := os.Getenv("LOG_LEVEL"); logLevelEnv != "" {
		switch logLevelEnv {
		case "DEBUG", "debug":
			level = zapcore.DebugLevel
		case "INFO", "info":
			level = zapcore.InfoLevel
		case "WARN", "warn":
			level = zapcore.WarnLevel
		case "ERROR", "error":
			level = zapcore.ErrorLevel
		default:
			// Keep default INFO level if invalid value provided
			level = zapcore.InfoLevel
		}
	}

	// Create file-only core (no console output)
	fileCore := zapcore.NewCore(encoder, fileWriter, level)

	// Create logger with caller information and stack trace on error
	logger := zap.New(fileCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	logInstance = &Service{
		logger: logger,
	}

	return logInstance, func() {
		logger.Sync()
		auxLogFile.Close()
	}, nil
}

// GetLogger returns the zap logger instance
func (s *Service) GetLogger() *zap.Logger {
	return s.logger
}

// Close flushes any buffered log entries
func (s *Service) Close() error {
	if s.logger != nil {
		return s.logger.Sync()
	}
	return nil
}

// Global convenience functions for easier access
var globalLogger *zap.Logger

// InitGlobalLogger initializes the global logger instance
func InitGlobalLogger() (func(), error) {
	service, closeFn, err := New()
	if err != nil {
		return nil, err
	}
	globalLogger = service.GetLogger()
	return closeFn, nil
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *zap.Logger {
	return globalLogger
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Debug(msg, fields...)
	}
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Info(msg, fields...)
	}
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Warn(msg, fields...)
	}
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Error(msg, fields...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Fatal(msg, fields...)
	}
}

// GetAuxLogger returns the auxiliary logger singleton instance
func GetAuxLogger() *log.Logger {
	return auxLogger
}

// AuxLog logs a message using the auxiliary logger
func AuxLog(msg string) {
	if auxLogger != nil {
		auxLogger.Println(msg)
	}
}

// AuxLogf logs a formatted message using the auxiliary logger
func AuxLogf(format string, v ...interface{}) {
	if auxLogger != nil {
		auxLogger.Printf(format, v...)
	}
}
