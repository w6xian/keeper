package cmd

import (
	"fmt"

	"github.com/w6xian/keeper"
	"github.com/w6xian/keeper/service"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}

var server_name = "TestService4"

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "注册为系统服务（开机自启）",
	RunE: func(cmd *cobra.Command, args []string) error {
		binPath := keeper.GetCaller()
		svc := service.New(server_name, "Go Keeper server3")
		if err := svc.Install(binPath, "abcd4"); err != nil {
			return fmt.Errorf("注册服务失败: %w", err)
		}
		fmt.Println("系统服务已注册，隧道将开机自启")
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "卸载系统服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.New(server_name, "Go Keeper server")
		if err := svc.Uninstall(); err != nil {
			return fmt.Errorf("卸载服务失败: %w", err)
		}
		fmt.Println("系统服务已卸载")
		return nil
	},
}
