package keeper

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"

	"github.com/w6xian/keeper/config"
	"github.com/w6xian/keeper/logger"
	"github.com/w6xian/keeper/service"

	"github.com/w6xian/sloth"
	"go.uber.org/zap"
)

type Door struct {
	logger  *zap.Logger
	svrConn *sloth.Connect
	addr    string
	wsPath  string
	wg      *sync.WaitGroup
	Name    string
}

func NewDoor(wg *sync.WaitGroup, options ...DoorOption) *Door {
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
		logger: logger.GetLogger(),
		Name:   ".door",
	}
	for _, opt := range options {
		opt(d)
	}

	// 1. Get random port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		d.logger.Fatal("Failed to listen", zap.Error(err))
	}
	port := ln.Addr().(*net.TCPAddr).Port
	d.addr = fmt.Sprintf("127.0.0.1:%d", port)
	d.wsPath = "/ws"
	// 2. Start Sloth Server
	// Create server logic container (ClientRpc handles server-side logic for incoming clients)
	clientRpc := sloth.DefaultServer()
	// Create connection manager
	d.svrConn = sloth.ServerConn(clientRpc)

	// Register RPC Service
	if err := d.svrConn.RegisterRpc("command", service.NewCommand(wg), ""); err != nil {
		d.logger.Fatal("Failed to register RPC", zap.Error(err))
	}
	// Register Log Service
	if err := d.svrConn.RegisterRpc("log", new(service.LogService), ""); err != nil {
		d.logger.Fatal("Failed to register Log RPC", zap.Error(err))
	}
	// Register Registry Service
	if err := d.svrConn.RegisterRpc("registry", service.NewRegistryService(), ""); err != nil {
		d.logger.Fatal("Failed to register Registry RPC", zap.Error(err))
	}
	// Register Script Service
	if err := d.svrConn.RegisterRpc("script", service.NewScriptService(), ""); err != nil {
		d.logger.Fatal("Failed to register Script RPC", zap.Error(err))
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
	d.svrConn.Listen("tcp", d.addr)
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

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

func (d *Door) Stop() error {
	pidFile := pidFilePath(d.Name)
	if err := os.Remove(pidFile); err != nil {
		logger.GetLogger().Error("failed to remove pid file %s: %w", zap.String("pidFile", pidFile), zap.Error(err))
		return err
	}
	return nil
}
