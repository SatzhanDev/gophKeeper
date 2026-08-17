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
const defaultHTTPPort = "8080"
const defaultTLSCertFile = "certs/server.crt"
const defaultTLSKeyFile = "certs/server.key"

// Config — конфигурация сервера GophKeeper.
type Config struct {
	DatabaseDSN string
	JWTSecret   string
	JWTTTL      time.Duration
	GRPCPort    string
	// HTTPPort — порт REST/Swagger-шлюза (grpc-gateway) поверх gRPC API.
	HTTPPort string
	// TLSCertFile, TLSKeyFile — путь к самоподписанному TLS-сертификату
	// и приватному ключу gRPC-сервера. Генерируются командой `make certs`.
	TLSCertFile string
	TLSKeyFile  string
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
	fs.StringVar(&cfg.HTTPPort, "hp", "", "HTTP port for REST/Swagger gateway (env: HTTP_PORT)")
	fs.StringVar(&cfg.TLSCertFile, "tls-cert", "", "path to TLS certificate (env: TLS_CERT_FILE)")
	fs.StringVar(&cfg.TLSKeyFile, "tls-key", "", "path to TLS private key (env: TLS_KEY_FILE)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Шаг 2: если флаг пуст — пробуем env. os.LookupEnv (а не os.Getenv)
	// используется намеренно: он возвращает второе значение ok, которое
	// явно отличает "переменная не задана" от "задана пустой строкой" —
	// os.Getenv в обоих случаях вернул бы "", и эти случаи было бы
	// невозможно различить.
	if cfg.DatabaseDSN == "" {
		if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
			cfg.DatabaseDSN = v
		}
	}
	if cfg.JWTSecret == "" {
		if v, ok := os.LookupEnv("JWT_SECRET"); ok {
			cfg.JWTSecret = v
		}
	}
	if cfg.JWTTTL == 0 {
		if v, ok := os.LookupEnv("JWT_TTL"); ok && v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, errors.New("invalid JWT_TTL: " + err.Error())
			}
			cfg.JWTTTL = d
		}
	}
	if cfg.GRPCPort == "" {
		if v, ok := os.LookupEnv("GRPC_PORT"); ok {
			cfg.GRPCPort = v
		}
	}
	if cfg.HTTPPort == "" {
		if v, ok := os.LookupEnv("HTTP_PORT"); ok {
			cfg.HTTPPort = v
		}
	}
	if cfg.TLSCertFile == "" {
		if v, ok := os.LookupEnv("TLS_CERT_FILE"); ok {
			cfg.TLSCertFile = v
		}
	}
	if cfg.TLSKeyFile == "" {
		if v, ok := os.LookupEnv("TLS_KEY_FILE"); ok {
			cfg.TLSKeyFile = v
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
	if cfg.GRPCPort == "" {
		cfg.GRPCPort = defaultGRPCPort
	}
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = defaultHTTPPort
	}
	if cfg.TLSCertFile == "" {
		cfg.TLSCertFile = defaultTLSCertFile
	}
	if cfg.TLSKeyFile == "" {
		cfg.TLSKeyFile = defaultTLSKeyFile
	}

	return cfg, nil
}
