package logger

import (
	"log/slog"
	"os"

	"gorm.io/gorm/logger"
)

// Logger 接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	GetGormLogger() logger.Interface
}

// slogger 基于slog的日志实现
type slogger struct {
	logger *slog.Logger
}

// New 创建新的日志器
func New(level string) Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &slogger{
		logger: logger,
	}
}

// Debug 调试日志
func (l *slogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info 信息日志
func (l *slogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn 警告日志
func (l *slogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error 错误日志
func (l *slogger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// Fatal 致命错误日志
func (l *slogger) Fatal(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

// GetGormLogger 获取GORM兼容的日志器
func (l *slogger) GetGormLogger() logger.Interface {
	return logger.Default.LogMode(logger.Info)
}