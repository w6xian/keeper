package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/service"
	"github.com/w6xian/keeper/utils/fsm"
)

var (
	token       string
	serviceName string
	configPath  string
	fsmType     string
	port        int
)

func init() {
	rootCmd.Flags().StringVar(&token, "token", "", "Token for the app websocket server")
	rootCmd.Flags().StringVar(&rootPath, "path", "", "Path of the root websocket server")
	rootCmd.Flags().StringVar(&serviceName, "service-name", server_name, "Windows service name")
	rootCmd.Flags().StringVar(&configPath, "config", "conf", "Path to the config file")
	rootCmd.Flags().StringVar(&fsmType, "fsm", "bolt", "Type of the FSM to use")
	rootCmd.Flags().IntVar(&port, "port", 8965, "Port of the app tcp server")

	rootCmd.Flags().Parse(os.Args)
}

var rootCmd = &cobra.Command{
	Use:   "keeper",
	Short: "Keeper is a lightweight process manager and script executor",
	Long:  `Keeper allows you to manage processes and execute scripts with ease.`,
	Run: func(cmd *cobra.Command, args []string) {
		runFunc := func(ctx context.Context) {
			// 只兜住 runFunc 自身的 panic：子 goroutine 里的 panic 仍会终止进程，
			// 它们的错误一律通过返回值/通道传回这里处理。
			defer func() {
				if r := recover(); r != nil {
					base := rootPath
					if base == "" {
						base = "."
					} else {
						base = filepath.Join(base, "keeper")
					}
					_ = os.MkdirAll(base, 0755)
					_ = os.WriteFile(filepath.Join(base, "crash.log"), debug.Stack(), 0644)
					log.Printf("panic recovered: %v", r)
				}
			}()

			wg := &sync.WaitGroup{}
			base := rootPath
			if base == "" {
				base = "."
			} else {
				base = filepath.Join(base, "data")
			}
			_ = os.MkdirAll(base, 0755)

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			fsmStore, err := fsm.NewFSM(fsmType, base)
			if err != nil {
				log.Printf("Failed to create FSM store: %v", err)
				return
			}
			defer func() {
				if err := fsmStore.Close(); err != nil {
					log.Printf("Failed to close FSM store: %v", err)
				}
			}()

			door := keeper.NewDoor(ctx, wg,
				keeper.WithDoorAddr("127.0.0.1:"+strconv.Itoa(port)),
				keeper.WithFSMStore(fsmStore),
			)

			// 监听失败必须能被感知：旧实现在 goroutine 里 log.Fatalf，
			// 端口被占用时进程直接消失，外面只知道"keeper 挂了"。
			serveErr := make(chan error, 1)
			go func() {
				serveErr <- door.Start()
			}()

			go func() {
				if _, err := door.ExecuteE(); err != nil {
					log.Printf("Child process exit: %v", err)
				}
			}()

			stopOnce := &sync.Once{}
			stop := func() {
				stopOnce.Do(func() {
					if err := door.Stop(); err != nil {
						log.Printf("Door stop failed: %v", err)
					}
				})
			}
			defer stop()

			wgDone := make(chan struct{})
			go func() {
				wg.Wait()
				close(wgDone)
			}()

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)
			defer signal.Stop(signalChan)

			select {
			case err := <-serveErr:
				if err != nil {
					log.Printf("Door serve stopped: %v", err)
				}
			case <-wgDone:
				log.Printf("All goroutines finished")
			case <-ctx.Done():
				log.Printf("Service stop requested")
			case <-signalChan:
				log.Printf("Shutting down...")
			}
		}

		// Try to run as service first
		if err := service.Run(serviceName, runFunc); err != nil {
			log.Printf("Service run failed: %v", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
