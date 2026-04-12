package api

import (
	"fmt"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	authHandler "lorem-backend/internal/modules/auth/handler"
	authRepo "lorem-backend/internal/modules/auth/repository"
	cartHandler "lorem-backend/internal/modules/cart/handler"
	cartRepo "lorem-backend/internal/modules/cart/repository"
	catHandler "lorem-backend/internal/modules/category/handler"
	catRepo "lorem-backend/internal/modules/category/repository"
	fileHandler "lorem-backend/internal/modules/file/handler"
	fileRepo "lorem-backend/internal/modules/file/repository"
	orderHandler "lorem-backend/internal/modules/order/handler"
	orderRepo "lorem-backend/internal/modules/order/repository"
	"lorem-backend/internal/modules/payment/gateway"
	paymentHandler "lorem-backend/internal/modules/payment/handler"
	paymentRepo "lorem-backend/internal/modules/payment/repository"
	productHandler "lorem-backend/internal/modules/product/handler"
	productRepo "lorem-backend/internal/modules/product/repository"
	userHandler "lorem-backend/internal/modules/user/handler"
	userRepo "lorem-backend/internal/modules/user/repository"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	loremMiddleware "lorem-backend/internal/api/middleware"
)

func NewRouter(db database.Database, s3 *s3.Client) *echo.Echo {
	router := echo.New()
	router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{config.GlobalConfig.FrontendURL},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
		AllowCredentials: true,
	}))
	router.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:  true,
		LogMethod:  true,
		LogURI:     true,
		LogError:   true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// Generate a readable timestamp: YYYY-MM-DD HH-MM-SS
			timestamp := time.Now().Format("2006-01-02 15:04:05")

			// Format: [TIMESTAMP] METHOD URI - STATUS
			if v.Error != nil {
				// Log with Error detail if something went wrong
				fmt.Printf("[%s] %s %s | STATUS: %d | LATENCY: %v ms | ERR: %v\n",
					timestamp, v.Method, v.URI, v.Status, v.Latency.Milliseconds(), v.Error)
			} else {
				// Standard Request Log
				fmt.Printf("[%s] %s %s | STATUS: %d | LATENCY: %v ms\n",
					timestamp, v.Method, v.URI, v.Status, v.Latency.Milliseconds())
			}

			return nil
		},
	}))

	registerAPIDocumentations(router)

	humaConfig := createHumaConfig()
	api := humaecho.New(router, humaConfig)

	// Setup Groups
	authGroup := huma.NewGroup(api, "/auth")
	protectedGroup := huma.NewGroup(api, "/api")

	// Apply verify token middleware to the rest
	protectedGroup.UseMiddleware(loremMiddleware.VerifyToken(api))

	// Register routes
	registerAuthRoute(authGroup, db)
	registerRoutes(protectedGroup, api, db, s3, router)

	return router
}

func registerAPIDocumentations(router *echo.Echo) {
	router.GET("/docs", StoplightElements)
}

func createHumaConfig() huma.Config {
	humaConfig := huma.DefaultConfig("Lorem E-Commerce", "1.0")
	return humaConfig
}

func registerRoutes(protected huma.API, public huma.API, db database.Database, s3 *s3.Client, e *echo.Echo) {
	// Init object storage repository
	s3Repository := fileRepo.NewS3Repository(s3)

	// Init file metadata repository
	fileRepository := fileRepo.NewFileMetaPostgresRepository(db, s3Repository)

	// Init product repository
	productRepository := productRepo.NewProductPostgresRepository(db)

	// Init order repository
	orderRepository := orderRepo.NewOrderPostgresRepository(db)

	registerUserRoute(protected, db)
	registerCategoryRoute(protected, db)
	registerProductRoute(protected, fileRepository, productRepository)
	registerFileRoute(protected, public, fileRepository)
	registerCartRoute(protected, db, fileRepository)
	registerOrderRoute(protected, orderRepository, productRepository)
	registerPaymentRoute(protected, e, db, orderRepository)
}

func registerAuthRoute(api huma.API, db database.Database) {
	// Init auth repo and handler
	authRepo := authRepo.NewAuthPostgresRepository(db)
	authHandler := authHandler.NewAuthHandlerImpl(authRepo)

	// POST /auth/register
	huma.Register(api, huma.Operation{
		OperationID:   "register-user",
		Method:        http.MethodPost,
		Path:          "/register",
		Summary:       "Register User",
		Description:   "Register a new user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusCreated,
	}, authHandler.RegisterUser)

	// POST /auth/signin
	huma.Register(api, huma.Operation{
		OperationID:   "sign-in-user",
		Method:        http.MethodPost,
		Path:          "/signin",
		Summary:       "Sign In User",
		Description:   "Sign in a user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, authHandler.SignInUser)

	// POST /auth/signout
	huma.Register(api, huma.Operation{
		OperationID:   "sign-out-user",
		Method:        http.MethodPost,
		Path:          "/signout",
		Summary:       "Sign Out User",
		Description:   "Sign out a user",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, authHandler.SignOutUser)
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

	// GET /user/me
	huma.Register(api, huma.Operation{
		OperationID:   "get-me",
		Method:        http.MethodGet,
		Path:          "/user/me",
		Summary:       "Get Me",
		Description:   "Get current user from session",
		Tags:          []string{"User"},
		DefaultStatus: http.StatusOK,
	}, userHandler.GetMe)
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

func registerProductRoute(api huma.API, file fileRepo.FileRepository, prodRepo productRepo.ProductRepository) {
	// Init product repo and handler
	productHandler := productHandler.NewProductHandlerImpl(prodRepo, file)

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

func registerFileRoute(protected huma.API, public huma.API, fileRepository fileRepo.FileRepository) {
	// Init file handler from repo
	fileHandler := fileHandler.NewFileHandlerImpl(fileRepository)

	// POST /file/upload
	huma.Register(protected, huma.Operation{
		OperationID:   "upload-file",
		Method:        http.MethodPost,
		Path:          "/file/upload",
		Summary:       "Upload File",
		Description:   "Upload a file to Object Storage",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusCreated,
	}, fileHandler.UploadFile)

	// POST /file/upload/static
	huma.Register(protected, huma.Operation{
		OperationID:   "upload-static-file",
		Method:        http.MethodPost,
		Path:          "/file/upload/static",
		Summary:       "Upload Static File",
		Description:   "Upload a file to Object Storage without modifying object key",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusCreated,
	}, fileHandler.UploadStaticFile)

	// GET /file/download/{id}
	huma.Register(protected, huma.Operation{
		OperationID:   "download-file",
		Method:        http.MethodGet,
		Path:          "/file/download/{id}",
		Summary:       "Download File",
		Description:   "Download a file from Object Storage",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusOK,
	}, fileHandler.DownLoadFile)

	// GET /file/download/key/{key} (public route mainly for downloading static image files)
	huma.Register(public, huma.Operation{
		OperationID:   "download-file-by-key",
		Method:        http.MethodGet,
		Path:          "/file/download/key/{key}",
		Summary:       "Download File By Key",
		Description:   "Download a file from Object Storage by object key",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusOK,
	}, fileHandler.DownloadFileByKey)

	// GET /file/{id}
	huma.Register(protected, huma.Operation{
		OperationID:   "get-file-metadata",
		Method:        http.MethodGet,
		Path:          "/file/{id}",
		Summary:       "Get File Metadata",
		Description:   "Get a file metadata from Object Storage",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusOK,
	}, fileHandler.GetFileMetaByID)

	// GET /file
	huma.Register(protected, huma.Operation{
		OperationID:   "get-all-files-metadata",
		Method:        http.MethodGet,
		Path:          "/file",
		Summary:       "Get Files Metadata",
		Description:   "Get all files metadata from Object Storage",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusOK,
	}, fileHandler.GetAllFilesMetadata)
}

func registerCartRoute(api huma.API, db database.Database, fileRepository fileRepo.FileRepository) {
	repo := cartRepo.NewCartPostgresRepository(db)
	handler := cartHandler.NewCartHandler(repo, fileRepository)

	// GET /user/{id}/cart
	huma.Register(api, huma.Operation{
		OperationID:   "get-user-cart",
		Method:        http.MethodGet,
		Path:          "/user/{id}/cart",
		Summary:       "Get User Cart",
		Description:   "Retrieve the active cart and items for a user",
		Tags:          []string{"Cart"},
		DefaultStatus: http.StatusOK,
	}, handler.GetCartByUserId)

	// POST /user/{id}/cart
	huma.Register(api, huma.Operation{
		OperationID:   "add-cart-item",
		Method:        http.MethodPost,
		Path:          "/user/{id}/cart",
		Summary:       "Add Item to Cart",
		Description:   "Add a new product to the cart or increase quantity if it exists",
		Tags:          []string{"Cart"},
		DefaultStatus: http.StatusOK,
	}, handler.CreateCartItem)

	// PUT /user/{id}/cart
	huma.Register(api, huma.Operation{
		OperationID:   "edit-cart-item",
		Method:        http.MethodPut,
		Path:          "/user/{id}/cart",
		Summary:       "Edit Cart Item",
		Description:   "Edit the exact quantity of a specific cart item (Must be >= 1)",
		Tags:          []string{"Cart"},
		DefaultStatus: http.StatusOK,
	}, handler.EditCartItem)

	// POST /user/{id}/cart/remove-items
	huma.Register(api, huma.Operation{
		OperationID:   "delete-cart-items",
		Method:        http.MethodPost,
		Path:          "/user/{id}/cart/remove-items",
		Summary:       "Remove Cart Items",
		Description:   "Remove one or multiple items from the cart using an array of Product IDs",
		Tags:          []string{"Cart"},
		DefaultStatus: http.StatusOK,
	}, handler.DeleteCartItems)
}

func registerOrderRoute(api huma.API, orderRepo orderRepo.OrderRepository, prodRepo productRepo.ProductRepository) {
	orderHandler := orderHandler.NewOrderHandlerImpl(orderRepo, prodRepo)

	// POST /order
	huma.Register(api, huma.Operation{
		OperationID:   "create-order",
		Method:        http.MethodPost,
		Path:          "/order",
		Summary:       "Create Order",
		Description:   "Create a new order from items",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusCreated,
	}, orderHandler.CreateOrder)

	// GET /user/{userId}/orders
	huma.Register(api, huma.Operation{
		OperationID:   "get-user-orders",
		Method:        http.MethodGet,
		Path:          "/user/{userId}/orders",
		Summary:       "Get User Orders",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusOK,
	}, orderHandler.GetOrders)

	// GET /order/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "get-order-by-id",
		Method:        http.MethodGet,
		Path:          "/order/{id}",
		Summary:       "Get Order By ID",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusOK,
	}, orderHandler.GetOrderById)

	// PATCH /order/{id}/status
	huma.Register(api, huma.Operation{
		OperationID:   "update-order-status",
		Method:        http.MethodPatch,
		Path:          "/order/{id}/status",
		Summary:       "Update Order Status",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusOK,
	}, orderHandler.UpdateOrderStatus)
}

func registerPaymentRoute(api huma.API, e *echo.Echo, db database.Database, orderRepo orderRepo.OrderRepository) {
	// Init Stripe gateway
	stripeGateway := gateway.NewStripePaymentGateway(config.GlobalConfig.StripeSecretKey, config.GlobalConfig.StripeWebhookSecret)

	// Init Payment Repository and Handler
	paymentRepository := paymentRepo.NewPaymentPostgresRepository(db)
	paymentHandler := paymentHandler.NewPaymentHandlerImpl(paymentRepository, orderRepo, stripeGateway)

	// Register Webhook directly via Echo
	// POST /api/webhook/stripe
	e.POST("/api/webhook/stripe", paymentHandler.HandleStripeWebhook)

	// Register standard API via huma
	// POST /api/payment/checkout
	huma.Register(api, huma.Operation{
		OperationID:   "create-checkout-session",
		Method:        http.MethodPost,
		Path:          "/payment/checkout",
		Summary:       "Create Checkout Session",
		Description:   "Create a Stripe checkout session for an order",
		Tags:          []string{"Payment"},
		DefaultStatus: http.StatusOK,
	}, paymentHandler.CreateCheckoutSession)
}
