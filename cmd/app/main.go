package main

import (
	"log"

	"go-cli-auth/internal/cli"
	"go-cli-auth/internal/config"
	"go-cli-auth/internal/db"
)

func main() {
	cfg := config.LoadConfig()

	database, err := db.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	app := cli.NewAppCLI(database, cfg)
	app.Start()
}