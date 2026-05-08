//go:build !windows

package service

import "context"

func Run(name string, handler func(ctx context.Context)) error {
	handler(context.Background())
	return nil
}
