package main

import (
	"log"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
)

func main() {
	config.LoadConfig()
	cfg := config.GlobalConfig
	db := database.NewPostgresDb(cfg)
	if err := database.SeedDatabase(db.GetDb()); err != nil {
		log.Fatal(err)
	}
	log.Println("Done!")
}
