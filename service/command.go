package service

import (
	"context"
	"fmt"
	"sync"
)

type Command struct {
	wg *sync.WaitGroup
}

func NewCommand(wg *sync.WaitGroup) *Command {
	return &Command{wg: wg}
}

func (s *Command) Exit(ctx context.Context, code int) ([]byte, error) {
	fmt.Printf("[Service] Exit called with: %d\n", code)
	s.wg.Done()
	return []byte("Exit " + fmt.Sprintf("%d", code)), nil
}
