package keeper

import (
	"fmt"
	"strings"
	"sync"

	"github.com/w6xian/keeper/internal/config"
)

// configMu 保护 initConfig：底层 viper 是进程级单例，
// 并发 Unmarshal 同一份全局变量会读到半初始化的 Conf，并产生数据竞争。
var configMu sync.Mutex

type Conf struct {
	LogMode  bool      `mapstructure:"log_mode"`
	Services []Service `mapstructure:"services"`
}

// initConfig 读取配置（支持 "a;b" 多路径，全部合并进同一个 viper 后统一解析）。
//
// 用 TryFromFiles 而不是 FromFiles：后者在文件缺失时直接 panic，
// 配置写错就把整个进程带崩，这在守护进程里不可接受。
func initConfig(f string) (*Conf, error) {
	configMu.Lock()
	defer configMu.Unlock()

	conf := &Conf{
		LogMode:  true,
		Services: []Service{},
	}

	var (
		parser  config.Unmarshal
		lastErr error
	)
	for _, item := range strings.Split(f, ";") {
		item = strings.TrimSuffix(strings.TrimSpace(item), ".toml")
		if item == "" {
			continue
		}
		p, err := config.TryFromFiles(item, config.TOML)
		if err != nil {
			lastErr = err
			continue
		}
		// 所有 Parser 读的是同一个全局 viper，取任意一个都能读到合并后的配置。
		parser = p
	}
	if parser == nil {
		return nil, fmt.Errorf("load config %q failed: %w", f, lastErr)
	}
	if err := parser.Unmarshal(conf); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	return conf, nil
}
