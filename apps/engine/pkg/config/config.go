package config

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds application configuration loaded from environment variables or config files.
type Config struct {
	AppEnv string `mapstructure:"APP_ENV" validate:"required,oneof=development staging production test"`
	//HTTPAddr        string        `mapstructure:"HTTP_ADDR" validate:"required,hostname_port|ip_port"`
	HTTPAddr        string        `mapstructure:"HTTP_ADDR" validate:"required,hostname_port"`
	ShutdownTimeout time.Duration `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`

	LogLevel  string `mapstructure:"LOG_LEVEL" validate:"required,oneof=debug info warn error dpanic panic fatal"`
	LogFormat string `mapstructure:"LOG_FORMAT" validate:"required,oneof=json console"`

	DatabaseURL string `mapstructure:"DATABASE_URL" validate:"required,url|uri"`

	//RedisAddr     string `mapstructure:"REDIS_ADDR" validate:"required,hostname_port|ip_port"`
	RedisAddr     string `mapstructure:"REDIS_ADDR" validate:"required,hostname_port"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`

	AsynqConcurrency int `mapstructure:"ASYNQ_CONCURRENCY" validate:"gte=1,lte=1000"`

	GoMaxProcs int `mapstructure:"GOMAXPROCS" validate:"gte=0,lte=4096"`
}

var (
	cfg      *Config
	validate = validator.New(validator.WithRequiredStructEnabled())
)

// Load initializes configuration using Viper. It loads from .env if present,
// applies defaults, binds env vars, and validates the result.
func Load() (*Config, error) {
	// Load .env if present (non-fatal)
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load()

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./apps/engine")

	// v.SetEnvPrefix("IAC")
	// v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("HTTP_ADDR", "0.0.0.0:8080")
	v.SetDefault("SHUTDOWN_TIMEOUT", "15s")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")
	v.SetDefault("ASYNQ_CONCURRENCY", 10)
	v.SetDefault("GOMAXPROCS", 0)

	// Optional config file
	_ = v.ReadInConfig()

	// Bind env without prefix for convenience
	keys := []string{
		"APP_ENV",
		"HTTP_ADDR",
		"SHUTDOWN_TIMEOUT",
		"LOG_LEVEL",
		"LOG_FORMAT",
		"DATABASE_URL",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"ASYNQ_CONCURRENCY",
		"GOMAXPROCS",
	}
	for _, key := range keys {
		_ = v.BindEnv(key)
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config unmarshal error: %w", err)
	}

	// Parse duration types that may come as string
	if s := v.GetString("SHUTDOWN_TIMEOUT"); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid SHUTDOWN_TIMEOUT: %w", err)
		}
		c.ShutdownTimeout = d
	}

	if err := validate.Struct(&c); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if c.GoMaxProcs > 0 {
		runtime.GOMAXPROCS(c.GoMaxProcs)
	}

	cfg = &c
	return cfg, nil
}

// MustLoad loads configuration or exits the process on failure.
func MustLoad() *Config {
	c, err := Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	return c
}

// Get returns the loaded configuration. Panics if not loaded.
func Get() *Config {
	if cfg == nil {
		panic("config not loaded: call config.Load or config.MustLoad first")
	}
	return cfg
}
