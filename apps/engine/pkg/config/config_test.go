package config

import (
	"os"
	"testing"
)

func TestWorkingDirBinding(t *testing.T) {
	// set required env vars for Load
	os.Setenv("APP_ENV", "test")
	os.Setenv("HTTP_ADDR", "127.0.0.1:8080")
	os.Setenv("SHUTDOWN_TIMEOUT", "1s")
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("LOG_FORMAT", "json")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/iac_test")
	os.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	os.Setenv("ASYNQ_CONCURRENCY", "1")
	os.Setenv("GOMAXPROCS", "1")

	// Set working dir
	tmp := t.TempDir()
	os.Setenv("WORKING_DIR", tmp)

	c, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if c.WorkingDir != tmp {
		t.Fatalf("expected working dir %s, got %s", tmp, c.WorkingDir)
	}
}
