package keeper

import (
	"fmt"
	"strings"
	"time"
)

func readOptions() (*Conf, error) {
	conf, err := initConfig("conf")
	if err != nil {
		return nil, err
	}
	if conf == nil {
		return nil, fmt.Errorf("initConfig failed")
	}
	return conf, nil
}

// Service 服务
// [[Service]]
// Name 服务名称
// name ="mili"
// Description 服务描述
// description = "mili-web-server __nginx"
// After 服务依赖的其他服务
// after = "network.target"
// Type 服务类型
// type = "forking"
// Start 启动时服务启动命令
// start = "/usr/local/nginx/sbin/nginx"
// Reload 重新加载时服务启动命令
// reload = "/usr/local/nginx/sbin/nginx -s reload"
// Stop 停止时服务启动命令
// stop =  "/usr/local/nginx/sbin/nginx -s quit"
type Service struct {
	Name               string `mapstructure:"name"`
	Description        string `mapstructure:"description"`
	After              string `mapstructure:"after"`
	Type               string `mapstructure:"type"`
	Start              string `mapstructure:"start"`
	Reload             string `mapstructure:"reload"`
	Stop               string `mapstructure:"stop"`
	RestartLimit       int    `mapstructure:"restart_limit"`
	RestartDelay       int    `mapstructure:"restart_delay"`
	RestartBackoffMax  int    `mapstructure:"restart_backoff_max"`
	RestartBackoffStep int    `mapstructure:"restart_backoff_step"`
	StopTimeout        int    `mapstructure:"stop_timeout"`
}

func (s *Service) ToToml() string {
	ts := []string{
		fmt.Sprintf("[[Service]]"),
		"# 服务名称",
		fmt.Sprintf("name = \"%s\"", s.Name),
		"# 服务描述",
		fmt.Sprintf("description = \"%s\"", s.Description),
		"# 服务依赖的其他服务",
		fmt.Sprintf("after = \"%s\"", s.After),
		"# 服务类型",
		fmt.Sprintf("type = \"%s\"", s.Type),
		"# 启动时服务启动命令",
		fmt.Sprintf("start = \"%s\"", s.Start),
		"# 重新加载时服务启动命令",
		fmt.Sprintf("reload = \"%s\"", s.Reload),
		"# 停止时服务启动命令",
		fmt.Sprintf("stop = \"%s\"", s.Stop),
		"# 异常退出后的最大重启次数",
		fmt.Sprintf("restart_limit = %d", s.EffectiveRestartLimit()),
		"# 每次重启前的基础等待(秒)",
		fmt.Sprintf("restart_delay = %d", s.EffectiveRestartDelaySeconds()),
		"# 退避等待的最大值(秒)",
		fmt.Sprintf("restart_backoff_max = %d", s.EffectiveRestartBackoffMaxSeconds()),
		"# 退避倍率(每次失败等待乘以该倍数)",
		fmt.Sprintf("restart_backoff_step = %d", s.EffectiveRestartBackoffStep()),
		"# 优雅停止超时时间(秒)",
		fmt.Sprintf("stop_timeout = %d", s.EffectiveStopTimeoutSeconds()),
	}
	return strings.Join(ts, "\n")
}

func (s Service) EffectiveRestartLimit() int {
	if s.RestartLimit < 0 {
		return 3
	}
	return s.RestartLimit
}

func (s Service) EffectiveRestartDelaySeconds() int {
	if s.RestartDelay <= 0 {
		return 1
	}
	return s.RestartDelay
}

func (s Service) EffectiveRestartBackoffMaxSeconds() int {
	if s.RestartBackoffMax <= 0 {
		return 60
	}
	return s.RestartBackoffMax
}

func (s Service) EffectiveRestartBackoffStep() int {
	if s.RestartBackoffStep <= 1 {
		return 2
	}
	return s.RestartBackoffStep
}

func (s Service) RestartBackoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	base := time.Duration(s.EffectiveRestartDelaySeconds()) * time.Second
	max := time.Duration(s.EffectiveRestartBackoffMaxSeconds()) * time.Second
	step := s.EffectiveRestartBackoffStep()

	d := base
	for i := 1; i < attempt; i++ {
		if d >= max {
			return max
		}
		d = d * time.Duration(step)
	}
	if d > max {
		return max
	}
	return d
}

func (s Service) EffectiveStopTimeout() time.Duration {
	return time.Duration(s.EffectiveStopTimeoutSeconds()) * time.Second
}

func (s Service) EffectiveStopTimeoutSeconds() int {
	if s.StopTimeout <= 0 {
		return 5
	}
	return s.StopTimeout
}
