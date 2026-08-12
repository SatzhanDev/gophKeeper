package main

import (
	"log"

	"github.com/SatzhanDev/gophKeeper/internal/client/cli"
	"github.com/SatzhanDev/gophKeeper/internal/client/config"
)

func main() {
	cfg := config.Load()

	if err := cli.Run(cfg.ServerAddr); err != nil {
		log.Fatal(err)
	}
}
