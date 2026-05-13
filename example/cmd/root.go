package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/spf13/cobra"
	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/logger"
	"github.com/w6xian/keeper/service"
	fsm "github.com/w6xian/keeper/utils/fsm"
	"go.uber.org/zap"
)

var (
	token       string
	serviceName string
)

func init() {
	rootCmd.Flags().StringVar(&token, "token", "", "Token for the app websocket server")
	rootCmd.Flags().StringVar(&rootPath, "path", "", "Path of the root websocket server")
	rootCmd.Flags().StringVar(&serviceName, "service-name", server_name, "Windows service name")

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

			dbDir := filepath.Join(base, "cache.db")
			opts := badger.DefaultOptions(dbDir)
			badgerDB, err := badger.Open(opts)
			if err != nil {
				_ = os.WriteFile(filepath.Join(base, "badger_open_error.log"), []byte(err.Error()), 0644)
				return
			}
			defer badgerDB.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fsmStore := fsm.NewBadger(badgerDB)
			door := keeper.NewDoor(ctx, wg, keeper.WithFSMStore(fsmStore), keeper.WithDoorAddr("127.0.0.1:8965"))
			go func() {
				err := door.Start()
				if err != nil {
					logger.GetLogger().Fatal("Failed to start dog", zap.Error(err))
				}
			}()

			// Wait a bit for server to start
			time.Sleep(200 * time.Millisecond)
			go door.Execute()
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
