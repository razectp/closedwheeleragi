package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Global zap logger instance
	globalLogger *zap.Logger
	globalSugar  *zap.SugaredLogger
)

// Init initializes the global zap logger
func Init(logLevel string, logFile string) error {
	// Parse log level
	var level zapcore.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = zapcore.DebugLevel
	case "INFO":
		level = zapcore.InfoLevel
	case "WARN":
		level = zapcore.WarnLevel
	case "ERROR":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create core
	var core zapcore.Core
	if logFile != "" {
		// File logging
		fileConfig := zap.Config{
			Level:       zap.NewAtomicLevelAt(level),
			Development: false,
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			Encoding:         "json",
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{logFile},
			ErrorOutputPaths: []string{logFile},
		}
		logger, err := fileConfig.Build()
		if err != nil {
			return err
		}
		globalLogger = logger
	} else {
		// Console logging
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		core = zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
		globalLogger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	}

	globalSugar = globalLogger.Sugar()
	return nil
}

// GetLogger returns the global zap logger
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		// Initialize with default config if not already done
		_ = Init("INFO", "")
	}
	return globalLogger
}

// GetSugarLogger returns the global sugared zap logger
func GetSugarLogger() *zap.SugaredLogger {
	if globalSugar == nil {
		// Initialize with default config if not already done
		_ = Init("INFO", "")
	}
	return globalSugar
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Convenience functions
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Sugared convenience functions
func Debugf(template string, args ...interface{}) {
	GetSugarLogger().Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	GetSugarLogger().Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	GetSugarLogger().Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	GetSugarLogger().Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	GetSugarLogger().Fatalf(template, args...)
}
