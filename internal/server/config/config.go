// Package config собирает конфигурацию сервера из флагов командной строки
// и переменных окружения. Переменные окружения имеют приоритет над флагами.
package config

import (
	"errors"
	"flag"
	"os"
	"time"
)

const defaultJWTTTL = 24 * time.Hour
const defaultGRPCPort = "50051"

// Config — конфигурация сервера GophKeeper.
type Config struct {
	DatabaseDSN string
	JWTSecret   string
	JWTTTL      time.Duration
	GRPCPort    string
}

// Load читает конфигурацию из аргументов командной строки args (обычно
// os.Args[1:]) в порядке приоритета:
// 1) флаги командной строки, 2) переменные окружения, 3) дефолтные значения.
//
// args принимается явным параметром (а не читается из os.Args внутри
// функции), чтобы Load можно было вызывать в юнит-тестах с разными
// наборами аргументов без побочных эффектов на глобальное состояние.
func Load(args []string) (*Config, error) {
	cfg := &Config{}

	// Шаг 1: флаги. Используем отдельный FlagSet, а не глобальный
	// flag.CommandLine — иначе повторный вызов Load (например, из разных
	// тестов) паникует с "flag redefined".
	fs := flag.NewFlagSet("gophkeeper-server", flag.ContinueOnError)
	fs.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN (env: DATABASE_DSN)")
	fs.StringVar(&cfg.JWTSecret, "j", "", "JWT signing secret (env: JWT_SECRET)")
	fs.DurationVar(&cfg.JWTTTL, "t", 0, "JWT TTL, e.g. 24h (env: JWT_TTL)")
	fs.StringVar(&cfg.GRPCPort, "gp", "", "GRPC port (env: GRPC_PORT)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

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
	if cfg.GRPCPort == "" {
		cfg.GRPCPort = os.Getenv("GRPC_PORT")
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
	if cfg.GRPCPort == "" {
		cfg.GRPCPort = defaultGRPCPort
	}

	return cfg, nil
}
