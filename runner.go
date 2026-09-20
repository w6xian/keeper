package keeper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

type serviceRuntime struct {
	service         Service
	cmd             *exec.Cmd
	exitCh          chan error
	restarts        int
	pendingRestart  bool
	nextRestartAt   time.Time
	stopRequested   bool
	restartDisabled bool
	mu              sync.Mutex
}

type keeperRunner struct {
	ctx      context.Context
	services []*serviceRuntime
	addr     string
	wsPath   string
	mu       sync.Mutex
}

func newKeeperRunner(ctx context.Context, services []Service, addr, wsPath string) *keeperRunner {
	runtimes := make([]*serviceRuntime, 0, len(services))
	for _, service := range services {
		runtimes = append(runtimes, &serviceRuntime{service: service})
	}
	return &keeperRunner{
		ctx:      ctx,
		services: runtimes,
		addr:     addr,
		wsPath:   wsPath,
	}
}

func (r *keeperRunner) run() error {
	started := make([]*serviceRuntime, 0, len(r.services))
	for _, runtime := range r.services {
		if err := r.start(runtime); err != nil {
			r.stopAll(started)
			return err
		}
		started = append(started, runtime)
		time.Sleep(time.Second)
	}
	if len(started) == 0 {
		log.Println("no service started")
		return nil
	}

	for {
		now := time.Now()
		select {
		case <-r.ctx.Done():
			r.stopAll(started)
			return r.ctx.Err()
		default:
		}

		for _, runtime := range started {
			if runtime.isRestartDisabled() {
				_, _, _ = runtime.pollExit()
				continue
			}
			if runtime.isRestartDue(now) {
				attempt := runtime.nextRestartCount()
				log.Printf("restarting service: %s, attempt=%d\n", runtime.service.Name, attempt)
				if restartErr := r.start(runtime); restartErr != nil {
					log.Printf("restart failed: %s, err=%v\n", runtime.service.Name, restartErr)
					runtime.scheduleRestart(time.Now(), attempt+1)
				}
				continue
			}

			exited, err, expected := runtime.pollExit()
			if !exited {
				continue
			}
			if expected {
				log.Printf("service exited: %s\n", runtime.service.Name)
				continue
			}
			if err == nil {
				log.Printf("service exited unexpectedly: %s\n", runtime.service.Name)
			} else {
				log.Printf("service exited: %s, err=%v\n", runtime.service.Name, err)
			}
			if runtime.isRestartDisabled() {
				continue
			}
			limit := runtime.service.EffectiveRestartLimit()
			if limit > 0 && runtime.restartCount() >= limit {
				log.Printf("service restart limit reached: %s, limit=%d\n", runtime.service.Name, runtime.service.EffectiveRestartLimit())
				continue
			}
			runtime.scheduleRestart(now, runtime.restartCount()+1)
		}
		time.Sleep(time.Second)
	}
}

func (r *keeperRunner) start(runtime *serviceRuntime) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.startLocked(runtime)
}

func (r *keeperRunner) startLocked(runtime *serviceRuntime) error {
	cmd, err := startService(r.ctx, runtime.service, r.addr, r.wsPath)
	if err != nil {
		return err
	}
	exitCh := make(chan error, 1)
	runtime.mu.Lock()
	runtime.cmd = cmd
	runtime.exitCh = exitCh
	runtime.stopRequested = false
	runtime.pendingRestart = false
	runtime.nextRestartAt = time.Time{}
	runtime.mu.Unlock()
	go func() {
		exitCh <- cmd.Wait()
	}()
	log.Printf("service started: %s, pid=%d\n", runtime.service.Name, cmd.Process.Pid)
	return nil
}

func (r *keeperRunner) StopService(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	target := strings.TrimSpace(name)
	if target == "" {
		return fmt.Errorf("service name is empty")
	}
	var runtime *serviceRuntime
	for _, item := range r.services {
		if strings.EqualFold(strings.TrimSpace(item.service.Name), target) {
			runtime = item
			break
		}
	}
	if runtime == nil {
		return fmt.Errorf("service not found: %s", target)
	}
	runtime.disableRestart()
	return stopService(runtime)
}

func (r *keeperRunner) StartService(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	target := strings.TrimSpace(name)
	if target == "" {
		return fmt.Errorf("service name is empty")
	}
	var runtime *serviceRuntime
	for _, item := range r.services {
		if strings.EqualFold(strings.TrimSpace(item.service.Name), target) {
			runtime = item
			break
		}
	}
	if runtime == nil {
		return fmt.Errorf("service not found: %s", target)
	}
	runtime.enableRestart()

	runtime.mu.Lock()
	running := runtime.cmd != nil && runtime.cmd.Process != nil
	runtime.mu.Unlock()
	if running {
		return nil
	}
	return r.startLocked(runtime)
}

func (r *keeperRunner) ReloadService(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	target := strings.TrimSpace(name)
	if target == "" {
		return fmt.Errorf("service name is empty")
	}
	var runtime *serviceRuntime
	for _, item := range r.services {
		if strings.EqualFold(strings.TrimSpace(item.service.Name), target) {
			runtime = item
			break
		}
	}
	if runtime == nil {
		return fmt.Errorf("service not found: %s", target)
	}
	runtime.enableRestart()

	runtime.mu.Lock()
	running := runtime.cmd != nil && runtime.cmd.Process != nil
	runtime.mu.Unlock()
	if !running {
		return r.startLocked(runtime)
	}

	reloadCmd := strings.TrimSpace(runtime.service.Reload)
	if reloadCmd == "" {
		if err := stopService(runtime); err != nil {
			return err
		}
		return r.startLocked(runtime)
	}

	reloadCtx, cancel := context.WithTimeout(context.Background(), runtime.service.EffectiveStopTimeout())
	defer cancel()
	reload := exec.CommandContext(reloadCtx, shellName(), shellArgs(reloadCmd)...)
	reload.Stdout = os.Stdout
	reload.Stderr = os.Stderr
	reload.Stdin = os.Stdin
	return reload.Run()
}

func (r *keeperRunner) stopAll(started []*serviceRuntime) {
	for i := len(started) - 1; i >= 0; i-- {
		runtime := started[i]
		if err := stopService(runtime); err != nil {
			log.Printf("stop service failed: %s, err=%v\n", runtime.service.Name, err)
		}
	}
}

func stopService(runtime *serviceRuntime) error {
	runtime.mu.Lock()
	cmd := runtime.cmd
	exitCh := runtime.exitCh
	runtime.stopRequested = true
	runtime.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if stopCmd := strings.TrimSpace(runtime.service.Stop); stopCmd != "" {
		stopCtx, cancel := context.WithTimeout(context.Background(), runtime.service.EffectiveStopTimeout())
		defer cancel()
		stop := exec.CommandContext(stopCtx, shellName(), shellArgs(stopCmd)...)
		stop.Stdout = os.Stdout
		stop.Stderr = os.Stderr
		stop.Stdin = os.Stdin
		if err := stop.Run(); err != nil {
			log.Printf("graceful stop command failed: %s, err=%v\n", runtime.service.Name, err)
		}
	}

	select {
	case <-time.After(runtime.service.EffectiveStopTimeout()):
		if err := cmd.Process.Kill(); err != nil && !isAlreadyDoneErr(err) {
			return err
		}
		if exitCh != nil {
			<-exitCh
		}
	case err := <-exitCh:
		if err != nil && !isExitErr(err) {
			return err
		}
	}

	// 进程确认退出后清空句柄：否则 StartService 看到 cmd != nil 会误判"还在跑"，
	// 直接返回成功却不启动任何东西（stop 之后再也 start 不起来）。
	runtime.mu.Lock()
	if runtime.cmd == cmd {
		runtime.cmd = nil
		runtime.exitCh = nil
		runtime.stopRequested = false
	}
	runtime.mu.Unlock()
	return nil
}

func (s *serviceRuntime) pollExit() (bool, error, bool) {
	s.mu.Lock()
	exitCh := s.exitCh
	s.mu.Unlock()
	if exitCh == nil {
		return false, nil, false
	}

	select {
	case err := <-exitCh:
		s.mu.Lock()
		expected := s.stopRequested
		s.cmd = nil
		s.exitCh = nil
		s.stopRequested = false
		s.mu.Unlock()
		return true, err, expected
	default:
		return false, nil, false
	}
}

func (s *serviceRuntime) scheduleRestart(now time.Time, attempt int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.restartDisabled {
		s.pendingRestart = false
		s.nextRestartAt = time.Time{}
		return
	}
	s.pendingRestart = true
	s.nextRestartAt = now.Add(s.service.RestartBackoffDuration(attempt))
}

func (s *serviceRuntime) isRestartDue(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.restartDisabled {
		return false
	}
	return s.pendingRestart && !s.nextRestartAt.IsZero() && !now.Before(s.nextRestartAt)
}

func (s *serviceRuntime) disableRestart() {
	s.mu.Lock()
	s.restartDisabled = true
	s.pendingRestart = false
	s.nextRestartAt = time.Time{}
	s.mu.Unlock()
}

// nextRestartCount 自增并返回重启次数（run 循环与 runner 的其它方法可能并发访问）。
func (s *serviceRuntime) nextRestartCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restarts++
	return s.restarts
}

func (s *serviceRuntime) restartCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restarts
}

func (s *serviceRuntime) isRestartDisabled() bool {
	s.mu.Lock()
	disabled := s.restartDisabled
	s.mu.Unlock()
	return disabled
}

func (s *serviceRuntime) enableRestart() {
	s.mu.Lock()
	s.restartDisabled = false
	s.pendingRestart = false
	s.nextRestartAt = time.Time{}
	s.restarts = 0
	s.mu.Unlock()
}

func sortServices(services []Service) ([]Service, error) {
	if len(services) == 0 {
		return nil, nil
	}

	index := make(map[string]Service, len(services))
	indegree := make(map[string]int, len(services))
	graph := make(map[string][]string, len(services))
	for _, service := range services {
		name := strings.TrimSpace(service.Name)
		if name == "" {
			return nil, fmt.Errorf("service name is empty")
		}
		if _, ok := index[name]; ok {
			return nil, fmt.Errorf("duplicate service name: %s", name)
		}
		index[name] = service
		indegree[name] = 0
	}

	for _, service := range services {
		for _, dep := range splitAfter(service.After) {
			if _, ok := index[dep]; !ok {
				return nil, fmt.Errorf("service %s depends on unknown service %s", service.Name, dep)
			}
			graph[dep] = append(graph[dep], service.Name)
			indegree[service.Name]++
		}
	}

	queue := make([]string, 0, len(services))
	for _, service := range services {
		if indegree[service.Name] == 0 {
			queue = append(queue, service.Name)
		}
	}

	result := make([]Service, 0, len(services))
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		result = append(result, index[name])
		for _, next := range graph[name] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(result) != len(services) {
		return nil, fmt.Errorf("service dependency cycle detected")
	}
	return result, nil
}

func splitAfter(after string) []string {
	if strings.TrimSpace(after) == "" {
		return nil
	}
	parts := strings.FieldsFunc(after, func(r rune) bool {
		return r == ',' || r == ';'
	})
	deps := make([]string, 0, len(parts))
	for _, part := range parts {
		dep := strings.TrimSpace(part)
		if dep == "" {
			continue
		}
		if !slices.Contains(deps, dep) {
			deps = append(deps, dep)
		}
	}
	return deps
}

func shellName() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "sh"
}

func shellArgs(commandLine string) []string {
	if runtime.GOOS == "windows" {
		return []string{"-Command", commandLine}
	}
	return []string{"-c", commandLine}
}

func isExitErr(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func isAlreadyDoneErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "process already finished")
}

// buildStartCommand 把服务启动命令和 keeper 注入的参数拼成 shell 要执行的一条命令。
//
// 参数必须拼进 shell 要执行的命令串里：sh -c "cmd --port X --path Y"。
// 旧写法是 append(shellArgs(commandLine), "--port", addr, "--path", wsPath)，
// 展开后是 sh -c "cmd" --port X --path Y —— 这两个参数会被 sh 当成脚本的
// 位置参数（$0 / $1），业务进程一个都收不到。
func buildStartCommand(commandLine, addr, wsPath string) string {
	return commandLine + " --port " + addr + " --path " + wsPath
}

func startService(ctx context.Context, service Service, addr, wsPath string) (*exec.Cmd, error) {
	commandLine := strings.TrimSpace(service.Start)
	if commandLine == "" {
		return nil, fmt.Errorf("empty start command")
	}
	full := buildStartCommand(commandLine, addr, wsPath)
	cmd := exec.CommandContext(ctx, shellName(), shellArgs(full)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
