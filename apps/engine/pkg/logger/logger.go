package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

// Init initializes a Zap logger with the provided level and format.
// level: debug, info, warn, error, dpanic, panic, fatal
// format: json, console
func Init(level, format string) (*zap.Logger, error) {
	lvl := zap.InfoLevel
	if err := lvl.Set(strings.ToLower(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.MessageKey = "message"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.TimeKey = "time"
	encoderCfg.LevelKey = "level"
	encoderCfg.CallerKey = "caller"
	encoderCfg.StacktraceKey = "stacktrace"
	encoderCfg.EncodeLevel = zapcore.LowercaseLevelEncoder

	var enc zapcore.Encoder
	switch strings.ToLower(format) {
	case "json":
		enc = zapcore.NewJSONEncoder(encoderCfg)
	case "console":
		enc = zapcore.NewConsoleEncoder(encoderCfg)
	default:
		return nil, fmt.Errorf("invalid log format %q", format)
	}

	core := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), lvl)
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	global = l
	return l, nil
}

// L returns the global logger. Panics if not initialized.
func L() *zap.Logger {
	if global == nil {
		panic("logger not initialized: call logger.Init first")
	}
	return global
}

// Sync flushes any buffered log entries.
func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}
