package logger

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/w6xian/keeper/internal/pathx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

// Config defines the logger configuration
type Config struct {
	Level      string
	Filename   string
	RootPath   string
	MaxSize    int // MB
	MaxBackups int
	MaxAge     int // Days
	Compress   bool
}

// InitLogger initializes the zap logger with lumberjack rotation
func InitLogger(cfg Config) error {
	// Ensure directory exists
	if cfg.RootPath != "" {
		dir := filepath.Dir(cfg.RootPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else {
		// Default if empty
		rootPath := pathx.GetCurrentAbPath()
		cfg.Filename = filepath.Join(rootPath, "/logs/keeper.log")
	}

	// Lumberjack logger for rotation
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize, // megabytes
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge, // days
		Compress:   cfg.Compress,
		LocalTime:  true, // Use local time for backup filenames
	})

	// Encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Parse level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		w,
		level,
	)

	// Add caller and stacktrace
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return nil
}

// GetLogger returns the global logger
var once sync.Once

func GetLogger() *zap.Logger {
	once.Do(func() {
		if Log == nil {
			// Fallback if not initialized
			l, _ := zap.NewProduction()
			Log = l
		}
	})
	return Log
}

// GenerateFilename generates a filename based on current date
func GenerateFilename(baseDir string) string {
	return filepath.Join(baseDir, time.Now().Format("2006-01-02")+".log")
}
