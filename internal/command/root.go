package command

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	"keeper/internal/config"
	"keeper/internal/logger"
	"keeper/internal/service"

	"github.com/spf13/cobra"
	"github.com/w6xian/sloth"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "keeper",
	Short: "A daemon process manager",
	Long:  `Keeper is a daemon process that manages an app process via WebSocket RPC.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO 确保 keeper守护程序只运行一次（写在一个特定文件，兼容windows，linux，macos）
		// TODO 写个文件，记录pid,然后独占地打开这个文件，文件不存在或能删除说明没有运行
		// TODO windows写在当前目录，linux/macos写在/var/run/keeper/keeper.pid
		runKeeper()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(AppCmd)
}

func runKeeper() {
	fmt.Printf("[Keeper] Starting daemon... PID: %d\n", os.Getpid())

	// 0. Load Config and Init Logger
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

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
	defer logger.GetLogger().Sync()

	logger.GetLogger().Info("Keeper started", zap.Int("pid", os.Getpid()))

	// 1. Get random port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		logger.GetLogger().Fatal("Failed to listen", zap.Error(err))
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // Release port for sloth
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	wsPath := "/ws"

	// 2. Start Sloth Server
	// Create server logic container (ClientRpc handles server-side logic for incoming clients)
	clientRpc := sloth.DefaultServer()
	// Create connection manager
	svrConn := sloth.ServerConn(clientRpc)

	// Register RPC Service
	if err := svrConn.RegisterRpc("keeper", new(service.HelloService), ""); err != nil {
		logger.GetLogger().Fatal("Failed to register RPC", zap.Error(err))
	}
	// Register Log Service
	if err := svrConn.RegisterRpc("log", new(service.LogService), ""); err != nil {
		logger.GetLogger().Fatal("Failed to register Log RPC", zap.Error(err))
	}
	// Register Registry Service
	if err := svrConn.RegisterRpc("registry", service.NewRegistryService(), ""); err != nil {
		logger.GetLogger().Fatal("Failed to register Registry RPC", zap.Error(err))
	}
	// Register Script Service
	if err := svrConn.RegisterRpc("script", service.NewScriptService(), ""); err != nil {
		logger.GetLogger().Fatal("Failed to register Script RPC", zap.Error(err))
	}

	// Start listening (in goroutine as it might block)
	// TODO 用waitgroup等待sloth启动完成
	var wg sync.WaitGroup
	defer wg.Done()
	go func() {
		wg.Add(1)
		// Note: Sloth's Listen might not return error based on doc, but let's check compilation
		svrConn.Listen("tcp", addr)
	}()
	wg.Wait()
	// TODO 写个文件，记录ws信息，YAML格式，包含addr,path

	// Wait a bit for server to start
	time.Sleep(200 * time.Millisecond)
	logger.GetLogger().Info("RPC Server listening", zap.String("addr", addr), zap.String("path", wsPath))
	// 3. Start Child Process (keeper app)
	exe, err := os.Executable()
	if err != nil {
		logger.GetLogger().Fatal("Failed to get executable path", zap.Error(err))
	}

	logger.GetLogger().Info("Launching 'app' subcommand", zap.String("exe", exe))

	cmd := exec.Command(exe, "app", "--port", addr, "--path", wsPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		logger.GetLogger().Fatal("Failed to start app process", zap.Error(err))
	}

	logger.GetLogger().Info("App process started", zap.Int("pid", cmd.Process.Pid))

	// 4. Wait for signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go func() {
		cmd.Wait()
		stop <- os.Interrupt
	}()

	<-stop
	logger.GetLogger().Info("Shutting down...")

	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		cmd.Process.Kill()
	}
}
