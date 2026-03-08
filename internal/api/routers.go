package api

import (
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	authHandler "lorem-backend/internal/modules/auth/handler"
	authRepo "lorem-backend/internal/modules/auth/repository"
	catHandler "lorem-backend/internal/modules/category/handler"
	catRepo "lorem-backend/internal/modules/category/repository"
	objectstorage "lorem-backend/internal/modules/objectStorage"
	productHandler "lorem-backend/internal/modules/product/handler"
	productRepo "lorem-backend/internal/modules/product/repository"
	userHandler "lorem-backend/internal/modules/user/handler"
	userRepo "lorem-backend/internal/modules/user/repository"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func NewRouter(db database.Database, cfg *config.Config, s3 *s3.Client) *echo.Echo {
	router := echo.New()
	router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	registerAPIDocumentations(router)

	humaConfig := createHumaConfig()
	api := humaecho.New(router, humaConfig)

	registerRoutes(api, db, cfg, s3)

	return router
}

func registerAPIDocumentations(router *echo.Echo) {
	router.GET("/docs", StoplightElements)
}

func createHumaConfig() huma.Config {
	humaConfig := huma.DefaultConfig("Lorem E-Commerce", "1.0")
	return humaConfig
}

func registerRoutes(api huma.API, db database.Database, cfg *config.Config, s3 *s3.Client) {
	// Init object storage repository
	s3Repo := objectstorage.NewS3Repository(s3, cfg)

	registerAuthRoute(api, db, cfg)
	registerUserRoute(api, db)
	registerCategoryRoute(api, db)
	registerProductRoute(api, db, s3Repo)
}

func registerAuthRoute(api huma.API, db database.Database, cfg *config.Config) {
	// Init auth repo and handler
	authRepo := authRepo.NewAuthPostgresRepository(db)
	authHandler := authHandler.NewAuthHandlerImpl(authRepo, cfg)

	// POST /auth/register
	huma.Register(api, huma.Operation{
		OperationID:   "register-user",
		Method:        http.MethodPost,
		Path:          "/auth/register",
		Summary:       "Register User",
		Description:   "Register a new user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusCreated,
	}, authHandler.RegisterUser)
}

func registerUserRoute(api huma.API, db database.Database) {
	// Init user repo and handler
	userRepo := userRepo.NewUserPostgresRepository(db)
	userHandler := userHandler.NewUserHandlerImpl(userRepo)

	// GET /user/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "get-user-by-id",
		Method:        http.MethodGet,
		Path:          "/user/{id}",
		Summary:       "Get User By ID",
		Description:   "Get a user by ID",
		Tags:          []string{"User"},
		DefaultStatus: http.StatusOK,
	}, userHandler.GetUserById)
}

func registerCategoryRoute(api huma.API, db database.Database) {
	// Init category repo and handler
	categoryRepo := catRepo.NewCategoryPostgresRepository(db)
	categoryHandler := catHandler.NewCategoryHandlerImpl(categoryRepo)

	// POST /category
	huma.Register(api, huma.Operation{
		OperationID:   "create-category",
		Method:        http.MethodPost,
		Path:          "/category",
		Summary:       "Create Category",
		Description:   "Create a new category",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusCreated,
	}, categoryHandler.CreateCategory)

	// GET /category/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "get-category-by-id",
		Method:        http.MethodGet,
		Path:          "/category/{id}",
		Summary:       "Get Category By ID",
		Description:   "Get a category by id",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusOK,
	}, categoryHandler.GetCategoryById)

	// GET /category
	huma.Register(api, huma.Operation{
		OperationID:   "get-categories",
		Method:        http.MethodGet,
		Path:          "/category",
		Summary:       "Get All Categories",
		Description:   "Get all categories",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusOK,
	}, categoryHandler.GetCategories)
}

func registerProductRoute(api huma.API, db database.Database, obj objectstorage.ObjectStorage) {
	// Init product repo and handler
	productRepo := productRepo.NewProductPostgresRepository(db)
	productHandler := productHandler.NewProductHandlerImpl(productRepo, obj)

	// POST /product
	huma.Register(api, huma.Operation{
		OperationID:   "create-product",
		Method:        http.MethodPost,
		Path:          "/product",
		Summary:       "Create Product",
		Description:   "Create a new product",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusCreated,
	}, productHandler.CreateProduct)

	// GET /product
	huma.Register(api, huma.Operation{
		OperationID:   "get-products",
		Method:        http.MethodGet,
		Path:          "/product",
		Summary:       "Get All Products",
		Description:   "Get all products",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusOK,
	}, productHandler.GetProducts)

	// GET /product/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "get-product-by-id",
		Method:        http.MethodGet,
		Path:          "/product/{id}",
		Summary:       "Get Product By ID",
		Description:   "Get product by ID",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusOK,
	}, productHandler.GetProductById)
}
