//go:build windows

package service

import (
	"golang.org/x/sys/windows/svc"
)

func Run(name string, handler func()) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if isService {
		return svc.Run(name, &serviceHandler{handler: handler})
	}
	handler()
	return nil
}

type serviceHandler struct {
	handler func()
}

func (m *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	s <- svc.Status{State: svc.StartPending}
	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	go m.handler()

loop:
	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			s <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break loop
		case svc.Pause:
			s <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
		case svc.Continue:
			s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
		}
	}
	s <- svc.Status{State: svc.StopPending}
	return false, 0
}
