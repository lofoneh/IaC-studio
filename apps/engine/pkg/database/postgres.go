package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iac-studio/engine/pkg/config"
	"github.com/iac-studio/engine/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// OpenPostgres opens a Gorm PostgreSQL connection with retry and sane pooling defaults.
func OpenPostgres(ctx context.Context, dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	logLevel := gormlogger.Silent
	if config.Get().AppEnv == "development" || config.Get().AppEnv == "test" {
		logLevel = gormlogger.Warn
	}

	b := backoff{
		maxRetries: 5,
		delay:      500 * time.Millisecond,
		maxDelay:   5 * time.Second,
	}

	for attempt := 0; ; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger2{zap: logger.L(), level: logLevel},
		})
		if err == nil {
			break
		}
		if attempt >= b.maxRetries {
			return nil, fmt.Errorf("open postgres failed after retries: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("open postgres canceled: %w", ctx.Err())
		case <-time.After(b.nextDelay(attempt)):
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("db db() error: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctxPing); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return db, nil
}

type logger2 struct {
	zap   *zap.Logger
	level gormlogger.LogLevel
}

func (l logger2) LogMode(level gormlogger.LogLevel) gormlogger.Interface { l.level = level; return l }
func (l logger2) Info(ctx context.Context, s string, args ...interface{}) {
	if l.level <= gormlogger.Info {
		l.zap.Sugar().Infof(s, args...)
	}
}
func (l logger2) Warn(ctx context.Context, s string, args ...interface{}) {
	if l.level <= gormlogger.Warn {
		l.zap.Sugar().Warnf(s, args...)
	}
}
func (l logger2) Error(ctx context.Context, s string, args ...interface{}) {
	if l.level <= gormlogger.Error {
		l.zap.Sugar().Errorf(s, args...)
	}
}
func (l logger2) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == gormlogger.Silent {
		return
	}
	sql, rows := fc()
	dur := time.Since(begin)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		l.zap.Error("gorm query error", zap.Duration("duration", dur), zap.Int64("rows", rows), zap.String("sql", sql), zap.Error(err))
		return
	}
	l.zap.Debug("gorm query", zap.Duration("duration", dur), zap.Int64("rows", rows), zap.String("sql", sql))
}

type backoff struct {
	maxRetries int
	delay      time.Duration
	maxDelay   time.Duration
}

func (b backoff) nextDelay(attempt int) time.Duration {
	d := b.delay << attempt
	if d > b.maxDelay {
		return b.maxDelay
	}
	return d
}
