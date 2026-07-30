package keeper

import (
	"fmt"
	"strings"
	"sync"

	"github.com/w6xian/keeper/internal/config"
)

var once sync.Once

type Conf struct {
	LogMode  bool      `mapstructure:"log_mode"`
	Services []Service `mapstructure:"services"`
}

func initConfig(f string) (*Conf, error) {
	conf := &Conf{
		LogMode:  true,
		Services: []Service{},
	}
	// 文件里读取配置
	cs := strings.Split(f, ";")
	var parser config.Unmarshal
	for _, f := range cs {
		// 去掉.toml后缀
		f = strings.TrimSuffix(f, ".toml")
		parser = config.FromFiles(f, config.TOML)
	}
	if err := parser.Unmarshal(conf); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	return conf, nil
}
