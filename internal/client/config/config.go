// Package config собирает конфигурацию клиента из флагов командной строки
// и переменных окружения. Переменные окружения имеют приоритет над флагами.
package config

import (
	"flag"
	"os"
)

const defaultServerAddr = "localhost:50051"
const defaultTLSCACertFile = "certs/server.crt"

// Config — конфигурация клиента GophKeeper.
type Config struct {
	ServerAddr  string
	ShowVersion bool
	// TLSCACertFile — путь к сертификату сервера, которому доверяет клиент
	// (сервер использует самоподписанный сертификат, поэтому обычный пул
	// системных CA его не примет — нужно явно указать, какому конкретному
	// сертификату доверять).
	TLSCACertFile string
}

// Load читает конфигурацию из аргументов командной строки args (обычно
// os.Args[1:]): сначала флаг, если пуст — переменная окружения, если и она
// пуста — встроенный дефолт.
//
// args принимается явным параметром, а флаги регистрируются в отдельном
// FlagSet (а не в глобальном flag.CommandLine), чтобы Load можно было
// безопасно вызывать многократно в юнит-тестах.
func Load(args []string) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("gophkeeper-client", flag.ContinueOnError)
	fs.StringVar(&cfg.ServerAddr, "server", "", "адрес gRPC-сервера (env: GOPHKEEPER_SERVER)")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "показать версию клиента и выйти")
	fs.StringVar(&cfg.TLSCACertFile, "tls-ca-cert", "", "path to trusted server TLS certificate (env: TLS_CA_CERT_FILE)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Шаг 2: если флаг пуст — пробуем env. os.LookupEnv, а не os.Getenv —
	// см. пояснение в internal/server/config/config.go.
	if cfg.ServerAddr == "" {
		if v, ok := os.LookupEnv("GOPHKEEPER_SERVER"); ok {
			cfg.ServerAddr = v
		}
	}
	if cfg.TLSCACertFile == "" {
		if v, ok := os.LookupEnv("TLS_CA_CERT_FILE"); ok {
			cfg.TLSCACertFile = v
		}
	}

	// Шаг 3: дефолт
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = defaultServerAddr
	}
	if cfg.TLSCACertFile == "" {
		cfg.TLSCACertFile = defaultTLSCACertFile
	}

	return cfg, nil
}
