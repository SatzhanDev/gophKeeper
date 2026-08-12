package main

import (
	"fmt"
	"log"

	"github.com/SatzhanDev/gophKeeper/internal/client/cli"
	"github.com/SatzhanDev/gophKeeper/internal/client/config"
	"github.com/SatzhanDev/gophKeeper/internal/pkg/version"
)

func main() {
	cfg := config.Load()

	if cfg.ShowVersion {
		fmt.Println(version.String("client"))
		return
	}

	if err := cli.Run(cfg.ServerAddr); err != nil {
		log.Fatal(err)
	}
}
