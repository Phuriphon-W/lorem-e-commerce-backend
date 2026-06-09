package main

import (
	"context"
	"log"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	fileRepo "lorem-backend/internal/modules/file/repository"
)

func main() {
	config.LoadConfig()
	cfg := config.GlobalConfig
	db := database.NewPostgresDb(cfg)

	s3Client, err := database.ConnectS3(cfg.S3Endpoint, cfg.AwsRegion, cfg.AwsAccessKey, cfg.AwsSecretKey)
	if err != nil {
		log.Fatal(err)
	}

	objectStorage := fileRepo.NewS3Repository(s3Client, nil)
	fileRepository := fileRepo.NewFileMetaPostgresRepository(db, objectStorage)

	if err := database.SeedDatabase(context.Background(), db.GetDb(), fileRepository); err != nil {
		log.Fatal(err)
	}
	log.Println("Done!")
}
