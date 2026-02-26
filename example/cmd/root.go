package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/logger"
	"github.com/w6xian/keeper/service"
	"go.uber.org/zap"
)

var (
	token string
)

func init() {
	rootCmd.Flags().StringVar(&token, "token", "", "Token for the app websocket server")

}

var rootCmd = &cobra.Command{
	Use:   "keeper",
	Short: "Keeper is a lightweight process manager and script executor",
	Long:  `Keeper allows you to manage processes and execute scripts with ease.`,
	Run: func(cmd *cobra.Command, args []string) {
		runFunc := func() {
			wg := &sync.WaitGroup{}
			door := keeper.NewDoor(wg)
			go func() {
				err := door.Start()
				if err != nil {
					logger.GetLogger().Fatal("Failed to start dog", zap.Error(err))
				}
			}()
			// Wait a bit for server to start
			time.Sleep(200 * time.Millisecond)
			go door.Execute()
			// 4. Wait for signals
			wg.Wait()
			logger.GetLogger().Info("Shutting down...")
			fmt.Println("wg out")
			door.Stop()
			os.Exit(0)
		}

		// Try to run as service first
		if err := service.Run("TestService3", runFunc); err != nil {
			logger.GetLogger().Fatal("Service run failed", zap.Error(err))
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
