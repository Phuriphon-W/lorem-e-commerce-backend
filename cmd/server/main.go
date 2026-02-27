package main

import (
	"fmt"
	"log"
	"net/http"

	"lorem-backend/internal/config"
	"lorem-backend/internal/database"

	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/labstack/echo/v4"
)

// Options defines the command-line options for the server.
// For example, go run cmd/server/main.go --p=2000 to start server on port 2000
type Options struct {
	Port int `doc:"Port to listen on." short:"p" default:"8888"`
}

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Connect to the database
	dbSession := database.ConnectDB(cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	defer database.DisconnectDB(dbSession)

	// Connect to S3
	s3Client, err := database.ConnectS3(cfg.S3Endpoint)
	if err != nil {
		log.Fatalf("Failed to connect to S3: %v", err)
	}

	fmt.Println(s3Client)

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		hooks.OnStart(func() {
			// Init Router
			router := echo.New()

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
