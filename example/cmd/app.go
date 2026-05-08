package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/utils/services"
)

var (
	appPort  string
	appPath  string
	rootPath string
)

func init() {
	appCmd.Flags().StringVar(&appPort, "port", "", "Address of the app websocket server")
	appCmd.Flags().StringVar(&appPath, "path", "", "Path of the app websocket server")
	appCmd.Flags().StringVar(&rootPath, "root", "", "Path of the root websocket server")
	rootCmd.AddCommand(appCmd)
}

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Run app",
	RunE: func(cmd *cobra.Command, args []string) error {
		if appPort == "" {
			return fmt.Errorf("port is required")
		}
		if appPath == "" {
			return fmt.Errorf("path is required")
		}
		ctx := context.Background()

		dog := keeper.NewDog(ctx, appPort, appPath)
		dog.InitService()
		go dog.KeepAlive()
		app := newApp()
		d, err := services.Get(ctx, "app")
		fmt.Println(d, err)
		if err != nil {
			services.Set(ctx, "app", []byte("app"))
		}
		// 这是keeper存储的app
		fmt.Println(string(d))
		app.Run(cmd, args)
		// keep run
		dog.Stop()
		os.Exit(0)
		return nil
	},
}

func newApp() *App {
	return &App{}
}

type App struct {
}

func (h *App) Run(cmd *cobra.Command, args []string) error {
	fmt.Println("App running -----------------")
	for {
		select {}
	}
}
