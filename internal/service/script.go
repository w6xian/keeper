package service

import (
	"context"
	"keeper/internal/logger"
	"sync"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

type ScriptService struct {
	Engine *lua.LState
	mu     sync.Mutex
}

func NewScriptService() *ScriptService {
	L := lua.NewState()
	return &ScriptService{
		Engine: L,
	}
}

func (s *ScriptService) Run(ctx context.Context, script string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	logger.GetLogger().Info("Executing Lua script", zap.String("script", script))
	if err := s.Engine.DoString(script); err != nil {
		logger.GetLogger().Error("Lua script execution failed", zap.Error(err))
		return "", err
	}
	return "Script executed successfully", nil
}

func (s *ScriptService) LoadFile(ctx context.Context, filename string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	logger.GetLogger().Info("Loading Lua file", zap.String("filename", filename))
	if err := s.Engine.DoFile(filename); err != nil {
		logger.GetLogger().Error("Lua file execution failed", zap.Error(err))
		return "", err
	}
	return "File executed successfully", nil
}
