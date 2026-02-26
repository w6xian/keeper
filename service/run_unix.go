//go:build !windows

package service

func Run(name string, handler func()) error {
	handler()
	return nil
}
