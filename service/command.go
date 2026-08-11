package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
)

type ProcController interface {
	StopService(ctx context.Context, name string) error
	StartService(ctx context.Context, name string) error
	ReloadService(ctx context.Context, name string) error
}

type Command struct {
	wg         *sync.WaitGroup
	controller ProcController
}

func NewCommand(wg *sync.WaitGroup, controller ProcController) *Command {
	return &Command{wg: wg, controller: controller}
}

func (s *Command) Exit(ctx context.Context, code int) ([]byte, error) {
	log.Printf("[Service] Exit called with: %d\n", code)
	s.wg.Done()
	return []byte("Exit " + fmt.Sprintf("%d", code)), nil
}

func (s *Command) KeepAlive(ctx context.Context, code int) ([]byte, error) {
	return []byte("KeepAlive " + fmt.Sprintf("%d", code)), nil
}

func (s *Command) StopService(ctx context.Context, name string) ([]byte, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return nil, fmt.Errorf("service name is empty")
	}
	if s.controller == nil {
		return nil, fmt.Errorf("controller is nil")
	}
	if err := s.controller.StopService(ctx, target); err != nil {
		return nil, err
	}
	return []byte("StopService " + target), nil
}

func (s *Command) StartService(ctx context.Context, name string) ([]byte, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return nil, fmt.Errorf("service name is empty")
	}
	if s.controller == nil {
		return nil, fmt.Errorf("controller is nil")
	}
	if err := s.controller.StartService(ctx, target); err != nil {
		return nil, err
	}
	return []byte("StartService " + target), nil
}

func (s *Command) ReloadService(ctx context.Context, name string) ([]byte, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return nil, fmt.Errorf("service name is empty")
	}
	if s.controller == nil {
		return nil, fmt.Errorf("controller is nil")
	}
	if err := s.controller.ReloadService(ctx, target); err != nil {
		return nil, err
	}
	return []byte("ReloadService " + target), nil
}
