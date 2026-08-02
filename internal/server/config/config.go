package config

import (
	"errors"
	"flag"
	"os"
	"time"
)

const defaultJWTTTL = 24 * time.Hour

// Config — конфигурация сервера GophKeeper.
type Config struct {
	DatabaseDSN string
	JWTSecret   string
	JWTTTL      time.Duration
}

// Load читает конфигурацию в порядке приоритета:
// 1) флаги командной строки, 2) переменные окружения, 3) дефолтные значения.
func Load() (*Config, error) {
	cfg := &Config{}

	// Шаг 1: флаги
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN (env: DATABASE_DSN)")
	flag.StringVar(&cfg.JWTSecret, "j", "", "JWT signing secret (env: JWT_SECRET)")
	flag.DurationVar(&cfg.JWTTTL, "t", 0, "JWT TTL, e.g. 24h (env: JWT_TTL)")
	flag.Parse()

	// Шаг 2: если флаг пуст — пробуем env
	if cfg.DatabaseDSN == "" {
		cfg.DatabaseDSN = os.Getenv("DATABASE_DSN")
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = os.Getenv("JWT_SECRET")
	}
	if cfg.JWTTTL == 0 {
		if v := os.Getenv("JWT_TTL"); v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, errors.New("invalid JWT_TTL: " + err.Error())
			}
			cfg.JWTTTL = d
		}
	}

	// Шаг 3: дефолты
	if cfg.JWTTTL == 0 {
		cfg.JWTTTL = defaultJWTTTL
	}
	if cfg.DatabaseDSN == "" {
		return nil, errors.New("database DSN is not set (use -d flag or DATABASE_DSN env)")
	}
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT secret is not set (use -j flag or JWT_SECRET env)")
	}

	return cfg, nil
}
