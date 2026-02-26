package service

import (
	"context"
	"fmt"

	"github.com/w6xian/keeper/logger"

	"go.uber.org/zap"
)

type LogService struct{}

func (s *LogService) Info(ctx context.Context, msg string) (string, error) {
	logger.GetLogger().Info(msg)
	return "ok", nil
}

func (s *LogService) Debug(ctx context.Context, msg string) (string, error) {
	logger.GetLogger().Debug(msg)
	return "ok", nil
}

func (s *LogService) Warn(ctx context.Context, msg string) (string, error) {
	logger.GetLogger().Warn(msg)
	return "ok", nil
}

func (s *LogService) Error(ctx context.Context, msg string) (string, error) {
	logger.GetLogger().Error(msg)
	return "ok", nil
}

// LogRequest is a struct for complex log requests if needed
type LogRequest struct {
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// Log handles dynamic logging
func (s *LogService) Log(ctx context.Context, req LogRequest) (string, error) {
	l := logger.GetLogger()
	// Add fields if any
	// zap.Any is simplified here
	var fields []zap.Field
	for k, v := range req.Fields {
		fields = append(fields, zap.Any(k, v))
	}

	switch req.Level {
	case "debug":
		l.Debug(req.Message, fields...)
	case "info":
		l.Info(req.Message, fields...)
	case "warn":
		l.Warn(req.Message, fields...)
	case "error":
		l.Error(req.Message, fields...)
	default:
		l.Info(fmt.Sprintf("[UNKNOWN LEVEL %s] %s", req.Level, req.Message), fields...)
	}
	return "ok", nil
}
