package command

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"keeper/internal/service"

	"github.com/spf13/cobra"
	"github.com/w6xian/sloth"
)

var rootCmd = &cobra.Command{
	Use:   "keeper",
	Short: "A daemon process manager",
	Long:  `Keeper is a daemon process that manages an app process via WebSocket RPC.`,
	Run: func(cmd *cobra.Command, args []string) {
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

	// 1. Get random port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
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
		log.Fatalf("Failed to register RPC: %v", err)
	}

	// Start listening (in goroutine as it might block)
	go func() {
		// Note: Sloth's Listen might not return error based on doc, but let's check compilation
		svrConn.Listen("tcp", addr)
	}()

	// Wait a bit for server to start
	time.Sleep(200 * time.Millisecond)

	fmt.Printf("[Keeper] RPC Server listening on %s (Path: %s)\n", addr, wsPath)

	// 3. Start Child Process (keeper app)
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("[Keeper] Failed to get executable path: %v", err)
	}

	fmt.Printf("[Keeper] Launching 'app' subcommand: %s app --port %s --path %s\n", exe, addr, wsPath)

	cmd := exec.Command(exe, "app", "--port", addr, "--path", wsPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		log.Fatalf("[Keeper] Failed to start app process: %v", err)
	}

	fmt.Printf("[Keeper] App process started with PID: %d\n", cmd.Process.Pid)

	// 4. Wait for signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go func() {
		cmd.Wait()
		stop <- os.Interrupt
	}()

	<-stop
	fmt.Println("[Keeper] Shutting down...")

	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		cmd.Process.Kill()
	}
}
