package main

import (
	"log"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
)

func main() {
	cfg := config.LoadConfig()
	db := database.NewPostgresDb(cfg)
	if err := database.SeedDatabase(db.GetDb()); err != nil {
		log.Fatal(err)
	}
	log.Println("Done!")
}
