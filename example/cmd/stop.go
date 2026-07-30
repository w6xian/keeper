package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/utils/services"
)

func init() {
	stopCmd.Flags().StringVar(&appPort, "port", "", "Address of the app websocket server")
	stopCmd.Flags().StringVar(&appPath, "path", "", "Path of the app websocket server")
	stopCmd.Flags().StringVar(&rootPath, "root", "", "Path of the root websocket server")
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().Parse(os.Args)
}

var stopCmd = &cobra.Command{
	Use:   "stop <service-name>",
	Short: "Stop service",
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

		dog := keeper.NewDog(ctx, addr, pth, keeper.WithDogName("stop-cli"))
		dog.InitService()
		if err := dog.KeepAlive(); err != nil {
			return err
		}
		resp, err := services.StopService(ctx, serviceName)
		_ = dog.Close()
		if err != nil {
			return err
		}
		fmt.Println(string(resp))
		os.Exit(0)
		return nil
	},
}
