package main

import (
	"log"

	"github.com/fenco/trademate/services/api/internal/app"
	"github.com/fenco/trademate/services/api/internal/config"
)

func main() {
	cfg := config.Load()
	server := app.NewServer(cfg)

	log.Printf("TradeMate API listening on :%s", cfg.APIPort)
	if err := server.Run(":" + cfg.APIPort); err != nil {
		log.Fatal(err)
	}
}
