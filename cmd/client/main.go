// Command gophkeeper-client — CLI-клиент GophKeeper: интерактивный доступ
// к приватным данным, хранящимся на сервере, с шифрованием и расшифровкой
// на стороне клиента. Запускается без аргументов (интерактивный режим),
// либо с флагом -version для вывода версии и даты сборки.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/SatzhanDev/gophKeeper/internal/client/cli"
	"github.com/SatzhanDev/gophKeeper/internal/client/config"
	"github.com/SatzhanDev/gophKeeper/internal/pkg/version"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	if cfg.ShowVersion {
		fmt.Println(version.String("client"))
		return
	}

	if err := cli.Run(cfg.ServerAddr, cfg.TLSCACertFile); err != nil {
		slog.Error("client exited with error", "err", err)
		os.Exit(1)
	}
}
