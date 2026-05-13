package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/utils/services"
)

func init() {
	rootCmd.AddCommand(luaCmd)
}

var luaCmd = &cobra.Command{
	Use:   "lua",
	Short: "Run lua",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := "127.0.0.1:8965"
		appPath := "/ws"
		if appPort != "" {
			addr = appPort
		}
		ctx := context.Background()
		dog := keeper.NewDog(ctx, addr, appPath)
		dog.InitService()
		fmt.Println("KeepAlive")
		err := dog.KeepAlive()
		if err != nil {
			return err
		}
		fmt.Println("KeepAlive done")
		l := newLuax(dog)
		d, err := services.Get(ctx, "app")
		if err != nil {
			return err
		}
		// 这是keeper存储的app
		fmt.Println(string(d))
		l.Run(cmd, args)
		// keep run
		// dog.Stop()
		os.Exit(0)
		return nil
	},
}

func newLuax(dkeeper *keeper.Dog) *Luax {
	return &Luax{keeper: dkeeper}
}

type Luax struct {
	keeper *keeper.Dog
}

func (h *Luax) Run(cmd *cobra.Command, args []string) error {
	services.Run(cmd.Context(), "print('hello world')")
	return nil
}
