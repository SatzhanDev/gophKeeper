// Package config собирает конфигурацию клиента из флагов командной строки
// и переменных окружения. Переменные окружения имеют приоритет над флагами.
package config

import (
	"flag"
	"os"
)

const defaultServerAddr = "localhost:50051"

// Config — конфигурация клиента GophKeeper.
type Config struct {
	ServerAddr  string
	ShowVersion bool
}

// Load читает конфигурацию: сначала флаг, если пуст — переменная
// окружения, если и она пуста — встроенный дефолт.
func Load() *Config {
	cfg := &Config{}

	// Шаг 1: флаг
	flag.StringVar(&cfg.ServerAddr, "server", "", "адрес gRPC-сервера (env: GOPHKEEPER_SERVER)")
	flag.BoolVar(&cfg.ShowVersion, "version", false, "показать версию клиента и выйти")

	flag.Parse()

	// Шаг 2: если флаг пуст — пробуем env
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = os.Getenv("GOPHKEEPER_SERVER")
	}

	// Шаг 3: дефолт
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = defaultServerAddr
	}

	return cfg
}
