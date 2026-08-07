package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/logger"
	"github.com/w6xian/keeper/service"
	"github.com/w6xian/keeper/utils/fsm"
	"go.uber.org/zap"
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
	rootCmd.Flags().IntVar(&port, "port", 8965, "Port of the app websocket server")

	rootCmd.Flags().Parse(os.Args)
}

var rootCmd = &cobra.Command{
	Use:   "keeper",
	Short: "Keeper is a lightweight process manager and script executor",
	Long:  `Keeper allows you to manage processes and execute scripts with ease.`,
	Run: func(cmd *cobra.Command, args []string) {
		runFunc := func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					// base := os.Getenv("PROGRAMDATA")
					base := rootPath
					if base == "" {
						base = "."
					} else {
						base = filepath.Join(base, "keeper")
					}
					_ = os.MkdirAll(base, 0755)
					_ = os.WriteFile(filepath.Join(base, "crash.log"), debug.Stack(), 0644)
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

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fsmStore, err := fsm.NewFSM(fsmType, base)
			if err != nil {
				logger.GetLogger().Fatal("Failed to create FSM store", zap.Error(err))
			}
			defer fsmStore.Close()
			door := keeper.NewDoor(ctx, wg, keeper.WithDoorAddr("127.0.0.1:"+strconv.Itoa(port)), keeper.WithFSMStore(fsmStore))
			go func() {
				err := door.Start()
				if err != nil {
					logger.GetLogger().Fatal("Failed to start dog", zap.Error(err))
				}
			}()
			// Wait a bit for server to start
			time.Sleep(200 * time.Millisecond)
			go door.TryExecuteFromConfig(configPath)
			// go door.Execute()

			stopOnce := &sync.Once{}
			stop := func() {
				stopOnce.Do(func() {
					door.Stop()
				})
			}

			wgDone := make(chan struct{})
			go func() {
				wg.Wait()
				close(wgDone)
			}()

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)
			select {
			case <-wgDone:
				logger.GetLogger().Info("All goroutines finished")
			case <-ctx.Done():
				logger.GetLogger().Info("Service stop requested")
			case <-signalChan:
				logger.GetLogger().Info("Shutting down...")
			}
			stop()
		}

		// Try to run as service first
		if err := service.Run(serviceName, runFunc); err != nil {
			logger.GetLogger().Fatal("Service run failed", zap.Error(err))
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
