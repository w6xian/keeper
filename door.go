package keeper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/w6xian/keeper/config"
	"github.com/w6xian/keeper/logger"
	"github.com/w6xian/keeper/service"
	"github.com/w6xian/keeper/utils/fsm"

	"github.com/w6xian/sloth/v3"
	"github.com/w6xian/sloth/v3/option"
	"go.uber.org/zap"
)

type Door struct {
	ctx      context.Context
	logger   *zap.Logger
	svrConn  *sloth.Connect
	addr     string
	wsPath   string
	wg       *sync.WaitGroup
	Name     string
	fsmStore fsm.IFSM
	runnerMu sync.Mutex
	runner   *keeperRunner
	childMu  sync.Mutex
	childCmd *exec.Cmd
}

func NewDoor(ctx context.Context, wg *sync.WaitGroup, options ...DoorOption) *Door {
	wg.Add(1)
	loggerConfig := logger.Config{
		Level:      config.GlobalConfig.Log.Level,
		Filename:   config.GlobalConfig.Log.Filename,
		MaxSize:    config.GlobalConfig.Log.MaxSize,
		MaxBackups: config.GlobalConfig.Log.MaxBackups,
		MaxAge:     config.GlobalConfig.Log.MaxAge,
		Compress:   config.GlobalConfig.Log.Compress,
	}

	if err := logger.InitLogger(loggerConfig); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}

	logger.GetLogger().Info("Dog started", zap.Int("pid", os.Getpid()))

	d := &Door{
		ctx:    ctx,
		logger: logger.GetLogger(),
		wg:     wg,
		Name:   ".door",
	}
	for _, opt := range options {
		opt(d)
	}
	if d.addr == "" {
		d.addr = "127.0.0.1:8965"
	}

	// 1. Get random port
	// ln, err := net.Listen("tcp", ":0")
	// if err != nil {
	// 	d.logger.Fatal("Failed to listen", zap.Error(err))
	// }
	// port := ln.Addr().(*net.TCPAddr).Port
	// d.addr = fmt.Sprintf("127.0.0.1:%d", port)

	d.wsPath = "/ws"
	// 2. Start Sloth Server
	// Create server logic container (ClientRpc handles server-side logic for incoming clients)
	clientRpc := sloth.DefaultServer()
	// Create connection manager
	d.svrConn = sloth.ServerConn(clientRpc)

	// Register RPC Service
	if err := d.svrConn.Register("command", service.NewCommand(wg, d), ""); err != nil {
		d.logger.Fatal("Failed to register RPC", zap.Error(err))
	}
	// Register Log Service
	if err := d.svrConn.Register("log", new(service.LogService), ""); err != nil {
		d.logger.Fatal("Failed to register Log RPC", zap.Error(err))
	}
	// Register Registry Service
	if err := d.svrConn.Register("registry", service.NewRegistryService(), ""); err != nil {
		d.logger.Fatal("Failed to register Registry RPC", zap.Error(err))
	}
	// Register Script Service
	if err := d.svrConn.Register("script", service.NewScriptService(), ""); err != nil {
		d.logger.Fatal("Failed to register Script RPC", zap.Error(err))
	}
	if d.fsmStore != nil {
		// Register Cache Service
		if err := d.svrConn.Register("cache", service.NewCache(d.fsmStore), ""); err != nil {
			d.logger.Fatal("Failed to register Cache RPC", zap.Error(err))
		}
	}

	return d
}

func (d *Door) Start() error {
	pidFile := pidFilePath(d.Name)
	pidManager := NewPIDManager(pidFile)
	if err := pidManager.WritePID(); err != nil {
		d.logger.Fatal("Failed to write PID file", zap.Error(err))
		os.Exit(1)
	}
	wsr := mux.NewRouter()
	if err := d.svrConn.Listen(d.ctx, "ws", d.addr,
		option.WithRouter(wsr, d.wsPath),
		option.WithServerHandleMessage(&ServerHandler{})); err != nil {
		return err
	}
	// http.Handle("/ws", wsr)
	if err := d.svrConn.Serve(); err != nil {
		return err
	}
	return nil
}

func (d *Door) Execute(args ...string) string {
	// Default: keeper app
	exe, err := os.Executable()
	if err != nil {
		logger.GetLogger().Fatal("Failed to get executable path", zap.Error(err))
	}
	cmdName := exe
	cmdArgs := []string{}
	if len(args) > 0 {
		cmdArgs = append(cmdArgs, args...)
	} else {
		cmdArgs = append(cmdArgs, "app")
	}
	// Append port and path arguments
	finalArgs := append(cmdArgs, "--port", d.addr, "--path", d.wsPath)
	// fmt.Println(cmdName, finalArgs)
	cmd := exec.Command(cmdName, finalArgs...)
	d.childMu.Lock()
	d.childCmd = cmd
	d.childMu.Unlock()
	if os.Getenv("KEEPER_SERVICE") != "1" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
	}

	if err := cmd.Start(); err != nil {
		logger.GetLogger().Fatal("Failed to start child process", zap.Error(err))
	}
	// fmt.Println("------start")
	if err := cmd.Wait(); err != nil {
		logger.GetLogger().Fatal("Child process exited with error", zap.Error(err))
	}
	// fmt.Println("------wait")
	return d.addr
}

func (d *Door) TryExecuteFromConfig(c string) error {
	conf, err := initConfig(c)
	if err != nil {
		logger.GetLogger().Error("failed to init config: %w", zap.Error(err))
		return err
	}
	ordered, err := sortServices(conf.Services)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(d.ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	runner := newKeeperRunner(ctx, ordered, d.addr, d.wsPath)
	d.setRunner(runner)
	defer d.setRunner(nil)
	if err := runner.run(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (d *Door) StopService(ctx context.Context, name string) error {
	d.runnerMu.Lock()
	runner := d.runner
	d.runnerMu.Unlock()
	if runner == nil {
		return fmt.Errorf("runner not started")
	}
	return runner.StopService(name)
}

func (d *Door) StartService(ctx context.Context, name string) error {
	d.runnerMu.Lock()
	runner := d.runner
	d.runnerMu.Unlock()
	if runner == nil {
		return fmt.Errorf("runner not started")
	}
	return runner.StartService(name)
}

func (d *Door) ReloadService(ctx context.Context, name string) error {
	d.runnerMu.Lock()
	runner := d.runner
	d.runnerMu.Unlock()
	if runner == nil {
		return fmt.Errorf("runner not started")
	}
	return runner.ReloadService(name)
}

func (d *Door) setRunner(runner *keeperRunner) {
	d.runnerMu.Lock()
	d.runner = runner
	d.runnerMu.Unlock()
}

func (d *Door) Stop() error {
	d.childMu.Lock()
	child := d.childCmd
	d.childCmd = nil
	d.childMu.Unlock()
	if child != nil && child.Process != nil {
		_ = child.Process.Kill()
	}
	pidFile := pidFilePath(d.Name)
	if err := os.Remove(pidFile); err != nil {
		logger.GetLogger().Error("failed to remove pid file %s: %w", zap.String("pidFile", pidFile), zap.Error(err))
		return err
	}
	return nil
}
