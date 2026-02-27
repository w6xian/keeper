package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/w6xian/keeper/internal/pathx"
)

type Config struct {
	Log LogConfig `mapstructure:"log"`
	App AppConfig `mapstructure:"app"`
}

type AppConfig struct {
	Command string   `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
}

var GlobalConfig Config

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // Days
	Compress   bool   `mapstructure:"compress"`
}

func LoadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	rootPath := pathx.GetCurrentAbPath()
	// Defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.filename", filepath.Join(rootPath, "/logs/keeper.log"))
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	viper.SetDefault("log.compress", true)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults
		fmt.Println("Config file not found, using defaults")
	}

	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}
