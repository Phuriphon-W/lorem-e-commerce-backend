package api

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func NewRouter() *echo.Echo {
	router := echo.New()
	router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	registerAPIDocumentations(router)

	humaConfig := createHumaConfig()
	api := humaecho.New(router, humaConfig)

	// TODO: Register routes using huma api
	fmt.Println(api)

	return router
}

func registerAPIDocumentations(router *echo.Echo) {
	router.GET("/docs", StoplightElements)
}

func createHumaConfig() huma.Config {
	humaConfig := huma.DefaultConfig("Lorem E-Commerce", "1.0")
	return humaConfig
}
