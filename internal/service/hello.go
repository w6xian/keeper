package service

import (
	"context"
	"fmt"
)

type HelloService struct{}

func (s *HelloService) SayHello(ctx context.Context, name string) (string, error) {
	fmt.Printf("[Service] SayHello called with: %s\n", name)
	return "Hello " + name, nil
}
