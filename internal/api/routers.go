package api

import (
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/user/handlers"
	"lorem-backend/internal/modules/user/repositories"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func NewRouter(db database.Database) *echo.Echo {
	router := echo.New()
	router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	registerAPIDocumentations(router)

	humaConfig := createHumaConfig()
	api := humaecho.New(router, humaConfig)

	registerRoutes(api, db)

	return router
}

func registerAPIDocumentations(router *echo.Echo) {
	router.GET("/docs", StoplightElements)
}

func createHumaConfig() huma.Config {
	humaConfig := huma.DefaultConfig("Lorem E-Commerce", "1.0")
	return humaConfig
}

func registerRoutes(api huma.API, db database.Database) {
	// Init user repo and handler
	userRepo := repositories.NewUserPostgresRepository(db)
	userHandler := handlers.NewUserHandlerImpl(userRepo)

	// POST /user
	huma.Register(api, huma.Operation{
		OperationID:   "create-user",
		Method:        http.MethodPost,
		Path:          "/user",
		Summary:       "Create User",
		Description:   "Create a new user",
		Tags:          []string{"User"},
		DefaultStatus: http.StatusCreated,
	}, userHandler.CreateUser)
}
