package main

import (
	"fmt"
	"log"
	"net/http"

	"lorem-backend/internal/api"
	"lorem-backend/internal/cache"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"

	"github.com/danielgtaylor/huma/v2/humacli"
)

// Options defines the command-line options for the server.
// For example, go run cmd/server/main.go --p=2000 to start server on port 2000
type Options struct {
	Port int `doc:"Port to listen on." short:"p" default:"8888"`
}

func main() {
	// Load configuration
	config.LoadConfig()
	cfg := config.GlobalConfig

	// Connect to the database
	db := database.NewPostgresDb(cfg)
	defer db.DisconnectDB()

	// Connect to S3
	s3Client, err := database.ConnectS3(cfg.S3Endpoint, cfg.AwsRegion, cfg.AwsAccessKey, cfg.AwsSecretKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to S3: %v", err))
	}

	// Connect to Redis (graceful fallback to nil if it fails/offline)
	var redisCache cache.Cache
	redisCache, err = cache.NewRedisCache(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Continuing in uncached mode.\n", err)
		redisCache = nil
	} else {
		log.Println("Connected to Redis successfully")
	}

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {

		// Create a new router and register APIs (from internal/api)
		router := api.NewRouter(db, s3Client, redisCache)

		hooks.OnStart(func() {
			port := cfg.Port
			if options.Port != 8888 {
				port = options.Port
			}
			log.Printf("Starting server on port %v...\n", port)
			log.Printf("API documentation is hosted at http://localhost:%d/docs\n", port)
			http.ListenAndServe(fmt.Sprintf(":%d", port), router)
		})
	})

	cli.Run()
}
