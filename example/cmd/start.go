package cmd

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/utils/services"
)

func init() {
	startCmd.Flags().StringVar(&appPort, "port", "", "Address of the app websocket server")
	startCmd.Flags().StringVar(&appPath, "path", "", "Path of the app websocket server")
	startCmd.Flags().StringVar(&rootPath, "root", "", "Path of the root websocket server")
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().Parse(os.Args)
}

var startCmd = &cobra.Command{
	Use:   "start <service-name>",
	Short: "Start service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		addr := "127.0.0.1:8965"
		pth := "/ws"
		if appPort != "" {
			addr = appPort
		}
		if appPath != "" {
			pth = appPath
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		dog := keeper.NewDog(ctx, addr, pth, keeper.WithDogName("start-cli"))
		dog.InitService()
		if err := dog.KeepAlive(); err != nil {
			return err
		}
		resp, err := services.StartService(ctx, serviceName)
		_ = dog.Close()
		if err != nil {
			return err
		}
		log.Println(string(resp))
		os.Exit(0)
		return nil
	},
}
