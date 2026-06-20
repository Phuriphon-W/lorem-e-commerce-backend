package api

import (
	"context"
	"fmt"
	"lorem-backend/internal/cache"
	"lorem-backend/internal/config"
	"lorem-backend/internal/database"
	authHandler "lorem-backend/internal/modules/auth/handler"
	authRepo "lorem-backend/internal/modules/auth/repository"
	cartHandler "lorem-backend/internal/modules/cart/handler"
	cartRepo "lorem-backend/internal/modules/cart/repository"
	catHandler "lorem-backend/internal/modules/category/handler"
	catRepo "lorem-backend/internal/modules/category/repository"
	emailService "lorem-backend/internal/modules/email/service"
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
	wsService "lorem-backend/internal/modules/websocket/service"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	loremMiddleware "lorem-backend/internal/api/middleware"

	"golang.org/x/time/rate"
)

const APIVersion = "/api/v1"

func NewRouter(db database.Database, s3 *s3.Client, redisCache cache.Cache) *echo.Echo {
	router := echo.New()
	router.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{config.GlobalConfig.FrontendURL},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE},
		AllowCredentials: true,
	}))

	// Rate limiting middleware for public auth endpoints
	rateLimitPeriod := time.Duration(config.GlobalConfig.RateLimitPeriodSec) * time.Second
	rateLimitLimit := config.GlobalConfig.RateLimitLimit

	rateLimiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			path := c.Path()
			return path != APIVersion+"/auth/signin" && path != APIVersion+"/auth/register" && path != APIVersion+"/auth/forgot-password"
		},
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(float64(rateLimitLimit) / rateLimitPeriod.Seconds()),
				Burst:     rateLimitLimit,
				ExpiresIn: rateLimitPeriod,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"message": "Too many requests"})
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"message": "Too many requests"})
		},
	})
	router.Use(rateLimiter)
	router.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:        true,
		LogMethod:        true,
		LogURI:           true,
		LogError:         true,
		LogLatency:       true,
		LogContentLength: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// Generate a readable timestamp: YYYY-MM-DD HH-MM-SS
			timestamp := time.Now().Format("2006-01-02 15:04:05")

			sizeInfo := ""
			if v.ContentLength != "" && v.ContentLength != "0" {
				sizeInfo = fmt.Sprintf(" | SIZE: %s", v.ContentLength)
			}

			// Format: [TIMESTAMP] METHOD URI - STATUS
			if v.Error != nil {
				// Log with Error detail if something went wrong
				fmt.Printf("[%s] %s %s | STATUS: %d | LATENCY: %v ms%s | ERR: %v\n",
					timestamp, v.Method, v.URI, v.Status, v.Latency.Milliseconds(), sizeInfo, v.Error)
			} else {
				// Standard Request Log
				fmt.Printf("[%s] %s %s | STATUS: %d | LATENCY: %v ms%s\n",
					timestamp, v.Method, v.URI, v.Status, v.Latency.Milliseconds(), sizeInfo)
			}

			return nil
		},
	}))

	registerAPIDocumentations(router)

	router.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	humaConfig := createHumaConfig()
	api := humaecho.New(router, humaConfig)

	// Setup Groups
	authGroup := huma.NewGroup(api, APIVersion+"/auth")
	protectedGroup := huma.NewGroup(api, APIVersion)
	publicApiGroup := huma.NewGroup(api, APIVersion)

	// Apply verify token middleware to the rest
	protectedGroup.UseMiddleware(loremMiddleware.VerifyToken(api))

	// Register routes
	registerAuthRoute(authGroup, db)
	registerRoutes(protectedGroup, publicApiGroup, api, db, s3, redisCache, router)

	return router
}

func registerAPIDocumentations(router *echo.Echo) {
	router.GET("/docs", StoplightElements)
}

func createHumaConfig() huma.Config {
	humaConfig := huma.DefaultConfig("Lorem E-Commerce", "1.0")
	return humaConfig
}

func registerRoutes(protected huma.API, publicApi huma.API, publicRoot huma.API, db database.Database, s3 *s3.Client, redisCache cache.Cache, e *echo.Echo) {
	// Init ws service
	wsSvc := wsService.NewWebsocketService()
	go wsSvc.Run(context.Background())

	// Register ws route
	e.GET("/ws", wsSvc.WebsocketHandler)

	// Init object storage repository
	s3Repository := fileRepo.NewS3Repository(s3, redisCache)

	// Init file metadata repository
	fileRepository := fileRepo.NewFileMetaPostgresRepository(db, s3Repository)

	// Init product repository
	productRepository := productRepo.NewProductPostgresRepository(db)

	// Init order repository
	orderRepository := orderRepo.NewOrderPostgresRepository(db)

	registerUserRoute(protected, db)
	registerCategoryRoute(protected, publicApi, db)
	registerProductRoute(protected, publicApi, fileRepository, productRepository)
	registerFileRoute(protected, publicRoot, fileRepository)
	registerCartRoute(protected, db, fileRepository, productRepository)
	registerOrderRoute(protected, db, orderRepository, productRepository, fileRepository)
	registerPaymentRoute(protected, e, db, orderRepository, productRepository, wsSvc)
}

func registerAuthRoute(api huma.API, db database.Database) {
	// Init email service
	emailSvc := emailService.NewSMTPEmailService(
		config.GlobalConfig.SmtpHost,
		config.GlobalConfig.SmtpPort,
		config.GlobalConfig.SmtpUser,
		config.GlobalConfig.SmtpPassword,
		config.GlobalConfig.SmtpFrom,
	)

	// Init auth repo and handler
	authRepo := authRepo.NewAuthPostgresRepository(db)
	authHandler := authHandler.NewAuthHandlerImpl(authRepo, emailSvc)

	// POST /auth/register
	huma.Register(api, huma.Operation{
		OperationID:   "register-user",
		Method:        http.MethodPost,
		Path:          "/register",
		Summary:       "Register User",
		Description:   "Creates a new customer account, hashes the password with bcrypt, persists the user in a transaction, and sets a signed JWT httpOnly cookie valid for the configured JWT_EXPIRE duration. Returns 409 if the email or username is already taken.",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusCreated,
	}, authHandler.RegisterUser)

	// POST /auth/signin
	huma.Register(api, huma.Operation{
		OperationID:   "sign-in-user",
		Method:        http.MethodPost,
		Path:          "/signin",
		Summary:       "Sign In User",
		Description:   "Validates credentials against the stored bcrypt hash and issues a signed JWT stored as an httpOnly cookie. Returns a generic 404 to prevent email enumeration. Rate-limited to 5 requests per 60 seconds per IP.",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, authHandler.SignInUser)

	// POST /auth/signout
	huma.Register(api, huma.Operation{
		OperationID:   "sign-out-user",
		Method:        http.MethodPost,
		Path:          "/signout",
		Summary:       "Sign Out User",
		Description:   "Clears the authToken httpOnly cookie by setting it to empty with MaxAge=-1, effectively ending the session without a server-side token store.",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, authHandler.SignOutUser)

	// POST /auth/forgot-password
	huma.Register(api, huma.Operation{
		OperationID:   "forgot-password",
		Method:        http.MethodPost,
		Path:          "/forgot-password",
		Summary:       "Forgot Password",
		Description:   "Accepts an email address and, if the account exists, asynchronously generates a 10-minute stateless JWT reset token and emails a reset link to the user. Always returns a generic success message to prevent email enumeration. Rate-limited to 5 requests per 60 seconds per IP.",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, authHandler.ForgotPassword)

	// POST /auth/reset-password
	huma.Register(api, huma.Operation{
		OperationID:   "reset-password",
		Method:        http.MethodPost,
		Path:          "/reset-password",
		Summary:       "Reset Password",
		Description:   "Verifies the stateless JWT reset token from the request body, extracts the user ID from its claims, hashes the new password with bcrypt, and updates it in the database. Returns 401 if the token is invalid or expired.",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusOK,
	}, authHandler.ResetPassword)
}

func registerUserRoute(api huma.API, db database.Database) {
	// Init user repo and handler
	userRepo := userRepo.NewUserPostgresRepository(db)
	userHandler := userHandler.NewUserHandlerImpl(userRepo)

	// GET /user
	huma.Register(api, huma.Operation{
		OperationID:   "get-all-users",
		Method:        http.MethodGet,
		Path:          "/user",
		Summary:       "Get All Users",
		Description:   "Returns a paginated list of all registered users with optional search and ordering. Requires admin privileges.",
		Tags:          []string{"User"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(api)},
	}, userHandler.GetUsers)

	// GET /user/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "get-user-by-id",
		Method:        http.MethodGet,
		Path:          "/user/{id}",
		Summary:       "Get User By ID",
		Description:   "Returns a single user's full profile including address fields. Requires the caller to be the account owner or an admin.",
		Tags:          []string{"User"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireOwnershipOrAdmin(api)},
	}, userHandler.GetUserById)

	// GET /user/me
	huma.Register(api, huma.Operation{
		OperationID:   "get-me",
		Method:        http.MethodGet,
		Path:          "/user/me",
		Summary:       "Get Me",
		Description:   "Returns the full profile of the currently authenticated user by reading the userID claim from the JWT cookie.",
		Tags:          []string{"User"},
		DefaultStatus: http.StatusOK,
	}, userHandler.GetMe)

	// PUT /user/me
	huma.Register(api, huma.Operation{
		OperationID:   "update-me",
		Method:        http.MethodPut,
		Path:          "/user/me",
		Summary:       "Update Me",
		Description:   "Updates the authenticated user's editable profile fields (first name, last name, telephone, and address). Fields not included in the request body are preserved. Requires a valid JWT session.",
		Tags:          []string{"User"},
		DefaultStatus: http.StatusCreated,
	}, userHandler.UpdateMe)

	// GET /user/count
	huma.Register(api, huma.Operation{
		OperationID:   "get-users-count",
		Method:        http.MethodGet,
		Path:          "/user/count",
		Summary:       "Get Users Count",
		Description:   "Returns the total number of registered users. Requires admin privileges.",
		Tags:          []string{"User"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(api)},
	}, userHandler.GetUsersCount)
}

func registerCategoryRoute(protected huma.API, publicApi huma.API, db database.Database) {
	// Init category repo and handler
	categoryRepo := catRepo.NewCategoryPostgresRepository(db)
	categoryHandler := catHandler.NewCategoryHandlerImpl(categoryRepo)

	// POST /category
	huma.Register(protected, huma.Operation{
		OperationID:   "create-category",
		Method:        http.MethodPost,
		Path:          "/category",
		Summary:       "Create Category",
		Description:   "Creates a new product category. Requires admin privileges. Returns 409 if the category name already exists.",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, categoryHandler.CreateCategory)

	// GET /category/{id}
	huma.Register(publicApi, huma.Operation{
		OperationID:   "get-category-by-id",
		Method:        http.MethodGet,
		Path:          "/category/{id}",
		Summary:       "Get Category By ID",
		Description:   "Returns a single category by its UUID. Public endpoint — no authentication required.",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusOK,
	}, categoryHandler.GetCategoryById)

	// GET /category
	huma.Register(publicApi, huma.Operation{
		OperationID:   "get-categories",
		Method:        http.MethodGet,
		Path:          "/category",
		Summary:       "Get All Categories",
		Description:   "Returns a list of all product categories. Public endpoint — no authentication required.",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusOK,
	}, categoryHandler.GetCategories)

	// PUT /category/{id}
	huma.Register(protected, huma.Operation{
		OperationID:   "update-category-by-id",
		Method:        http.MethodPut,
		Path:          "/category/{id}",
		Summary:       "Update Category By ID",
		Description:   "Updates the name or metadata of an existing category by its UUID. Requires admin privileges.",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, categoryHandler.UpdateCategory)

	// DELETE /category/{id}
	huma.Register(protected, huma.Operation{
		OperationID:   "delete-category-by-id",
		Method:        http.MethodDelete,
		Path:          "/category/{id}",
		Summary:       "Delete Category By ID",
		Description:   "Soft-deletes a category by its UUID using GORM's soft-delete mechanism. Requires admin privileges.",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, categoryHandler.DeleteCategory)

	// GET /category/count
	huma.Register(protected, huma.Operation{
		OperationID:   "get-categories-count",
		Method:        http.MethodGet,
		Path:          "/category/count",
		Summary:       "Get Categories Count",
		Description:   "Returns the total number of categories. Requires admin privileges.",
		Tags:          []string{"Category"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, categoryHandler.GetCategoriesCount)
}

func registerProductRoute(protected huma.API, publicApi huma.API, file fileRepo.FileRepository, prodRepo productRepo.ProductRepository) {
	// Init product repo and handler
	productHandler := productHandler.NewProductHandlerImpl(prodRepo, file)

	// POST /product
	huma.Register(protected, huma.Operation{
		OperationID:   "create-product",
		Method:        http.MethodPost,
		Path:          "/product",
		Summary:       "Create Product",
		Description:   "Creates a new product by uploading the image file to AWS S3 (stored under the product-images/ prefix) and persisting the product metadata and S3 object key in PostgreSQL. Requires admin privileges. Accepts multipart/form-data.",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, productHandler.CreateProduct)

	// GET /product
	huma.Register(publicApi, huma.Operation{
		OperationID:   "get-products",
		Method:        http.MethodGet,
		Path:          "/product",
		Summary:       "Get All Products",
		Description:   "Returns a paginated list of products with optional category filter, search term, and sort order (price_low, price_high, name_asc, name_desc, date_asc, or newest-first by default). Each product response includes a short-lived S3 presigned URL for the product image, served from Redis cache. Public endpoint.",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusOK,
	}, productHandler.GetProducts)

	// GET /product/{id}
	huma.Register(publicApi, huma.Operation{
		OperationID:   "get-product-by-id",
		Method:        http.MethodGet,
		Path:          "/product/{id}",
		Summary:       "Get Product By ID",
		Description:   "Returns full details for a single product by UUID, including a short-lived S3 presigned image URL served from Redis cache. Public endpoint.",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusOK,
	}, productHandler.GetProductById)

	// DELETE /product/{id}
	huma.Register(protected, huma.Operation{
		OperationID:   "delete-product-by-id",
		Method:        http.MethodDelete,
		Path:          "/product/{id}",
		Summary:       "Delete Product By ID",
		Description:   "Soft-deletes a product by UUID. The S3 image object is not removed. Requires admin privileges.",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, productHandler.DeleteProductById)

	// PUT /product/{id}
	huma.Register(protected, huma.Operation{
		OperationID:   "update-product-by-id",
		Method:        http.MethodPut,
		Path:          "/product/{id}",
		Summary:       "Update Product By ID",
		Description:   "Updates product metadata (name, description, price, availability, category). If a new image file is provided in the multipart form, it is uploaded to S3 and the object key is updated. Requires admin privileges.",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, productHandler.UpdateProduct)

	// GET /product/count
	huma.Register(protected, huma.Operation{
		OperationID:   "get-products-count",
		Method:        http.MethodGet,
		Path:          "/product/count",
		Summary:       "Get Products Count",
		Description:   "Returns the total number of products. Requires admin privileges.",
		Tags:          []string{"Product"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, productHandler.GetProductsCount)
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
		Description:   "Uploads a file to the configured S3-compatible object storage under a timestamped key and records the file metadata (object key, filename, size, content type) in PostgreSQL. Requires admin privileges. Accepts multipart/form-data.",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, fileHandler.UploadFile)

	// POST /file/upload/static
	huma.Register(protected, huma.Operation{
		OperationID:   "upload-static-file",
		Method:        http.MethodPost,
		Path:          "/file/upload/static",
		Summary:       "Upload Static File",
		Description:   "Uploads a file to S3 using the original filename as the object key (no timestamp prefix), suitable for static assets. Records metadata in PostgreSQL. Requires admin privileges.",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, fileHandler.UploadStaticFile)

	// GET /file/download/{id}
	huma.Register(protected, huma.Operation{
		OperationID:   "download-file",
		Method:        http.MethodGet,
		Path:          "/file/download/{id}",
		Summary:       "Download File",
		Description:   "Retrieves a file from S3 by its database UUID and streams it to the client. Requires a valid JWT session.",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusOK,
	}, fileHandler.DownLoadFile)

	// GET /file/download/key/{key} (public route mainly for downloading static image files)
	huma.Register(public, huma.Operation{
		OperationID:   "download-file-by-key",
		Method:        http.MethodGet,
		Path:          "/file/download/key/{key}",
		Summary:       "Download File By Key",
		Description:   "Generates and returns a short-lived S3 presigned URL for a file identified by its object key. Public endpoint — primarily used for serving static image assets.",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusOK,
	}, fileHandler.DownloadFileByKey)

	// GET /file/{id}
	huma.Register(protected, huma.Operation{
		OperationID:   "get-file-metadata",
		Method:        http.MethodGet,
		Path:          "/file/{id}",
		Summary:       "Get File Metadata",
		Description:   "Returns the stored metadata record (object key, filename, size, content type, timestamps) for a file by its UUID. Requires a valid JWT session.",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusOK,
	}, fileHandler.GetFileMetaByID)

	// GET /file
	huma.Register(protected, huma.Operation{
		OperationID:   "get-all-files-metadata",
		Method:        http.MethodGet,
		Path:          "/file",
		Summary:       "Get Files Metadata",
		Description:   "Returns all file metadata records stored in PostgreSQL. Requires admin privileges.",
		Tags:          []string{"File"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(protected)},
	}, fileHandler.GetAllFilesMetadata)
}

func registerCartRoute(api huma.API, db database.Database, fileRepository fileRepo.FileRepository, productRepository productRepo.ProductRepository) {
	repo := cartRepo.NewCartPostgresRepository(db)
	handler := cartHandler.NewCartHandler(repo, fileRepository, productRepository)

	// GET /user/{id}/cart
	huma.Register(api, huma.Operation{
		OperationID:   "get-user-cart",
		Method:        http.MethodGet,
		Path:          "/user/{id}/cart",
		Summary:       "Get User Cart",
		Description:   "Returns the active cart and all its items (including product details and presigned image URLs) for the specified user. Requires the caller to be the cart owner or an admin.",
		Tags:          []string{"Cart"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireOwnershipOrAdmin(api)},
	}, handler.GetCartByUserId)

	// POST /user/{id}/cart
	huma.Register(api, huma.Operation{
		OperationID:   "add-cart-item",
		Method:        http.MethodPost,
		Path:          "/user/{id}/cart",
		Summary:       "Add Item to Cart",
		Description:   "Adds a product to the cart. If the product already exists in the cart, its quantity is incremented. Validates that the requested quantity does not exceed available stock. Requires a customer role and ownership of the cart.",
		Tags:          []string{"Cart"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireCustomer(api), loremMiddleware.RequireOwnershipOrAdmin(api)},
	}, handler.CreateCartItem)

	// PUT /user/{id}/cart
	huma.Register(api, huma.Operation{
		OperationID:   "edit-cart-item",
		Method:        http.MethodPut,
		Path:          "/user/{id}/cart",
		Summary:       "Edit Cart Item",
		Description:   "Sets the exact quantity of a specific product in the cart. Quantity must be >= 1. Validates against current stock availability. Requires a customer role and ownership of the cart.",
		Tags:          []string{"Cart"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireCustomer(api), loremMiddleware.RequireOwnershipOrAdmin(api)},
	}, handler.EditCartItem)

	// POST /user/{id}/cart/remove-items
	huma.Register(api, huma.Operation{
		OperationID:   "delete-cart-items",
		Method:        http.MethodPost,
		Path:          "/user/{id}/cart/remove-items",
		Summary:       "Remove Cart Items",
		Description:   "Removes one or more items from the cart by product UUID. Accepts an array of product IDs in the request body. Requires a customer role and ownership of the cart.",
		Tags:          []string{"Cart"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireCustomer(api), loremMiddleware.RequireOwnershipOrAdmin(api)},
	}, handler.DeleteCartItems)
}

func registerOrderRoute(api huma.API, db database.Database, orderRepo orderRepo.OrderRepository, prodRepo productRepo.ProductRepository, fileRepo fileRepo.FileRepository) {
	orderHandler := orderHandler.NewOrderHandlerImpl(db, orderRepo, prodRepo, fileRepo)

	// POST /order
	huma.Register(api, huma.Operation{
		OperationID:   "create-order",
		Method:        http.MethodPost,
		Path:          "/order",
		Summary:       "Create Order",
		Description:   "Creates a new order from the specified cart items, deducting stock quantities using row-level locking (SELECT FOR UPDATE) within a database transaction to prevent overselling. Requires a customer role.",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusCreated,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireCustomer(api)},
	}, orderHandler.CreateOrder)

	// GET /user/{userId}/orders
	huma.Register(api, huma.Operation{
		OperationID:   "get-user-orders",
		Method:        http.MethodGet,
		Path:          "/user/{userId}/orders",
		Summary:       "Get User Orders",
		Description:   "Returns a paginated list of orders placed by the specified user, including order items with presigned product image URLs. Requires the caller to be the order owner or an admin.",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireOwnershipOrAdmin(api)},
	}, orderHandler.GetOrders)

	// GET /order/{id}
	huma.Register(api, huma.Operation{
		OperationID:   "get-order-by-id",
		Method:        http.MethodGet,
		Path:          "/order/{id}",
		Summary:       "Get Order By ID",
		Description:   "Returns full details of a single order by UUID, including all line items with presigned product image URLs. Requires a valid JWT session.",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusOK,
	}, orderHandler.GetOrderById)

	// PATCH /order/{id}/status
	huma.Register(api, huma.Operation{
		OperationID:   "update-order-status",
		Method:        http.MethodPatch,
		Path:          "/order/{id}/status",
		Summary:       "Update Order Status",
		Description:   "Updates the fulfillment status of an order (e.g., pending, processing, shipped, delivered, cancelled). Requires admin privileges.",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(api)},
	}, orderHandler.UpdateOrderStatus)

	// GET /order/count
	huma.Register(api, huma.Operation{
		OperationID:   "get-orders-count",
		Method:        http.MethodGet,
		Path:          "/order/count",
		Summary:       "Get Orders Count",
		Description:   "Returns the total number of orders across all users. Requires admin privileges.",
		Tags:          []string{"Order"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireAdmin(api)},
	}, orderHandler.GetOrdersCount)
}

func registerPaymentRoute(api huma.API, e *echo.Echo, db database.Database, orderRepo orderRepo.OrderRepository, productRepo productRepo.ProductRepository, wsSvc wsService.WebsocketService) {
	// Init Stripe gateway
	stripeGateway := gateway.NewStripePaymentGateway(config.GlobalConfig.StripeSecretKey, config.GlobalConfig.StripeWebhookSecret)

	// Init Payment Repository and Handler
	paymentRepository := paymentRepo.NewPaymentPostgresRepository(db)
	paymentHandler := paymentHandler.NewPaymentHandlerImpl(paymentRepository, orderRepo, productRepo, stripeGateway, wsSvc)

	// Register Webhook directly via Echo
	// POST /api/v1/webhook/stripe
	e.POST(APIVersion+"/webhook/stripe", paymentHandler.HandleStripeWebhook)

	// Register standard API via huma
	// POST /api/payment/checkout
	huma.Register(api, huma.Operation{
		OperationID:   "create-checkout-session",
		Method:        http.MethodPost,
		Path:          "/payment/checkout",
		Summary:       "Create Checkout Session",
		Description:   "Creates a Stripe Checkout Session for the specified order, setting the session expiry from STRIPE_SESSION_EXPIRE. Requires a customer role. Returns the Stripe-hosted checkout URL.",
		Tags:          []string{"Payment"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireCustomer(api)},
	}, paymentHandler.CreateCheckoutSession)

	// Get /api/payment/{userId}
	huma.Register(api, huma.Operation{
		OperationID:   "get-user-payments",
		Method:        http.MethodGet,
		Path:          "/payment/{userId}",
		Summary:       "Get User Payments",
		Description:   "Returns the payment history records (status, Stripe session ID, amount) for a specific user. Requires the caller to be the account owner or an admin.",
		Tags:          []string{"Payment"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{loremMiddleware.RequireOwnershipOrAdmin(api)},
	}, paymentHandler.GetUserPaymentsByUserID)

	// GET /api/payment/verify
	huma.Register(api, huma.Operation{
		OperationID:   "verify-session",
		Method:        http.MethodGet,
		Path:          "/payment/verify",
		Summary:       "Verify Session",
		Description:   "Verifies the status of a Stripe Checkout Session by session ID and returns whether the payment has been completed. Used by the frontend to confirm payment success after redirect.",
		Tags:          []string{"Payment"},
		DefaultStatus: http.StatusOK,
	}, paymentHandler.VerifySession)
}
