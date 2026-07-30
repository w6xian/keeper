package config

type Logger struct {
	FilePath    string `mapstructure:"file_path"`    // 日志文件路径
	Level       int8   `mapstructure:"level"`        // 日志级别
	MaxSize     int    `mapstructure:"max_size"`     // 每个日志文件保存的最大尺寸 单位：M
	MaxBackups  int    `mapstructure:"max_backups"`  // 日志文件最多保存多少个备份
	MaxAge      int    `mapstructure:"max_age"`      // 文件最多保存多少天
	Compress    bool   `mapstructure:"compress"`     // 是否压缩
	ServiceName string `mapstructure:"service_name"` // 服务名
	Stdout      bool   `mapstructure:"std_out"`
	Filename    string `mapstructure:"file_name"`
	LocalTime   bool   `mapstructure:"localtime"`
	Debug       bool   `mapstructure:"debug"`
}
