package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Service struct {
	logger *zap.Logger
}

var (
	logInstance    *Service
	auxLogger      *log.Logger
	auxLogFile     io.WriteCloser
	auxExtraWriter io.Writer // Extra writer for working zone
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

	logFile, err := tea.LogToFile(auxLogPath, "aux")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create aux log file: %v", err)
	}
	auxLogFile = logFile

	// Initialize auxiliary logger singleton if not already created
	if auxLogger == nil {
		// Initially write only to file, extra writer can be added later
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

// SetAuxLogWriter sets an additional writer for the auxiliary logger
// This is typically used to send aux logs to the working zone during spinner mode
func SetAuxLogWriter(writer io.Writer) {
	auxExtraWriter = writer

	if auxLogger != nil && auxLogFile != nil {
		if writer != nil {
			// Create a multi-writer that writes to both file and extra writer
			multiWriter := io.MultiWriter(auxLogFile, writer)
			auxLogger.SetOutput(multiWriter)
		} else {
			// Reset to file-only output
			auxLogger.SetOutput(auxLogFile)
		}
	}
}

// ClearAuxLogWriter removes the extra writer from aux logger
func ClearAuxLogWriter() {
	SetAuxLogWriter(nil)
}

// AuxLog logs a message using the auxiliary logger
func AuxLog(msg string) {
	if auxLogger != nil {
		auxLogger.Println(msg)
	}
}

// AuxLogf logs a formatted message using the auxiliary logger
func AuxLogf(format string, args ...interface{}) {
	if auxLogger != nil {
		auxLogger.Printf(format, args...)
	}
}

// LogPanic captures panic information and logs it to ~/.vastix/logs/panic.log
// This should be used as a deferred function at the top level of the application
// The panic will still propagate and crash the program after logging
func LogPanic() {
	if r := recover(); r != nil {
		// Get full stack trace (all goroutines)
		buf := make([]byte, 1024*64)  // 64KB buffer for full trace
		n := runtime.Stack(buf, true) // true = all goroutines
		stackTrace := string(buf[:n])

		// Try to log to panic.log file
		if vastixDir, err := GetVastixDir(); err == nil {
			logsDir := filepath.Join(vastixDir, "logs")
			panicLogPath := filepath.Join(logsDir, "panic.log")

			// Open panic log file in append mode
			if f, err := os.OpenFile(panicLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				panicInfo := fmt.Sprintf("\n"+
					"================================================================================\n"+
					"PANIC at %s\n"+
					"================================================================================\n"+
					"Error: %v\n\n"+
					"Stack Trace:\n%s\n"+
					"================================================================================\n\n",
					timestamp, r, stackTrace)

				f.WriteString(panicInfo)
				f.Close()

				// Print to stderr where the panic log was saved
				fmt.Fprintf(os.Stderr, "\nPanic details saved to: %s\n\n", panicLogPath)
			}
		}

		// Also log to main application logger if available
		if globalLogger != nil {
			globalLogger.Error("Application panic",
				zap.Any("panic", r),
				zap.String("stack_trace", stackTrace),
			)
			globalLogger.Sync() // Flush logs before crash
		}

		// Also log to aux logger if available
		if auxLogger != nil {
			auxLogger.Printf("PANIC: %v\n%s", r, stackTrace)
		}

		// Re-panic to let the program crash naturally
		panic(r)
	}
}
