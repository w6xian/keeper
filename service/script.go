package service

import (
	"context"
	"log"
	"sync"

	"github.com/w6xian/gua"
)

type ScriptService struct {
	Engine *gua.Luax
	mu     sync.Mutex
}

func NewScriptService() *ScriptService {
	L := gua.NewState(gua.CallStackSize(1024))
	return &ScriptService{
		Engine: L,
	}
}

func (s *ScriptService) Run(ctx context.Context, script string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("Executing Lua script %s", script)
	if err := s.Engine.DoString(script); err != nil {
		log.Printf("Lua script execution failed %v", err)
		return "", err
	}
	return "Script executed successfully", nil
}

func (s *ScriptService) LoadFile(ctx context.Context, filename string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("Loading Lua file %s", filename)
	if err := s.Engine.DoFile(filename); err != nil {
		log.Printf("Lua file execution failed %v", err)
		return "", err
	}
	return "File executed successfully", nil
}
