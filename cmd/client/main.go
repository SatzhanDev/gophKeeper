// Command gophkeeper-client — CLI-клиент GophKeeper: интерактивный доступ
// к приватным данным, хранящимся на сервере, с шифрованием и расшифровкой
// на стороне клиента. Запускается без аргументов (интерактивный режим),
// либо с флагом -version для вывода версии и даты сборки.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SatzhanDev/gophKeeper/internal/client/cli"
	"github.com/SatzhanDev/gophKeeper/internal/client/config"
	"github.com/SatzhanDev/gophKeeper/internal/pkg/version"
)

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if cfg.ShowVersion {
		fmt.Println(version.String("client"))
		return
	}

	if err := cli.Run(cfg.ServerAddr); err != nil {
		log.Fatal(err)
	}
}
