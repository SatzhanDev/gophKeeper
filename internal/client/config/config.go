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
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Шаг 2: если флаг пуст — пробуем env
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = os.Getenv("GOPHKEEPER_SERVER")
	}

	// Шаг 3: дефолт
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = defaultServerAddr
	}

	return cfg, nil
}
