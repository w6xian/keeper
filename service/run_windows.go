//go:build windows

package service

import (
	"context"
	"time"

	"golang.org/x/sys/windows/svc"
)

func Run(name string, handler func(ctx context.Context)) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	if isService {
		return svc.Run(name, &serviceHandler{handler: handler, cancel: cancel, ctx: ctx, done: make(chan struct{})})
	}
	handler(ctx)
	return nil
}

type serviceHandler struct {
	handler func(ctx context.Context)
	cancel  context.CancelFunc
	ctx     context.Context
	done    chan struct{}
}

func (m *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	s <- svc.Status{State: svc.StartPending}
	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	go func() {
		defer close(m.done)
		m.handler(m.ctx)
	}()

loop:
	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			s <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			m.cancel()
			break loop
		case svc.Pause:
			s <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
		case svc.Continue:
			s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
		}
	}
	s <- svc.Status{State: svc.StopPending}
	select {
	case <-m.done:
	case <-time.After(15 * time.Second):
	}
	return false, 0
}
