package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"fmbq-server/config"
	"fmbq-server/database"
	"fmbq-server/handlers"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

func main() {
	// Load configuration
	if err := config.Load(); err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Connect to database
	db, err := database.Connect(config.AppConfig.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize tables
	if err := db.InitializeTables(); err != nil {
		log.Fatal("Failed to initialize tables:", err)
	}

	// Initialize Cloudinary
	cloudinaryURL := config.AppConfig.CloudinaryURL
	if cloudinaryURL == "" {
		// Fallback to hardcoded URL if .env is not loaded
		cloudinaryURL = "cloudinary://967168151421182:ozy93r-T7tSlcEiJ79g9xfhe_N4@dt5vwrozu"
		log.Printf("Using hardcoded Cloudinary URL")
	}
	
	log.Printf("Initializing Cloudinary with URL: %s", cloudinaryURL)
	if err := services.InitializeCloudinary(cloudinaryURL); err != nil {
		log.Printf("ERROR: Failed to initialize Cloudinary: %v", err)
		log.Printf("Cloudinary URL: %s", cloudinaryURL)
	} else {
		log.Printf("SUCCESS: Cloudinary initialized successfully")
	}

	// Set Gin mode
	if config.AppConfig.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.Default()

	// Add CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure this properly for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "FMBQ Server is running",
		})
	})
	
	// Debug endpoint for network testing
	router.GET("/debug/network", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "network_test",
			"message": "Network connectivity test successful",
			"timestamp": time.Now().Unix(),
			"client_ip": c.ClientIP(),
		})
	})

	// Debug endpoint to check order_items table
	router.GET("/debug/order-items", func(c *gin.Context) {
		// Check if order_items table exists and has data
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM order_items").Scan(&count)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to query order_items table", "details": err.Error()})
			return
		}

		// Get sample data - first check table structure
		tableInfoQuery := `
			SELECT column_name, data_type 
			FROM information_schema.columns 
			WHERE table_name = 'order_items' 
			ORDER BY ordinal_position
		`
		tableRows, err := db.Query(tableInfoQuery)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get table info", "details": err.Error()})
			return
		}
		defer tableRows.Close()

		var columns []map[string]string
		for tableRows.Next() {
			var colName, dataType string
			err := tableRows.Scan(&colName, &dataType)
			if err != nil {
				continue
			}
			columns = append(columns, map[string]string{
				"name": colName,
				"type": dataType,
			})
		}

		// Get sample data - handle NULL values for UUID columns
		rows, err := db.Query("SELECT id, order_id, product_id, sku_id, quantity FROM order_items LIMIT 5")
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to query order_items data", "details": err.Error()})
			return
		}
		defer rows.Close()

		var items []map[string]interface{}
		for rows.Next() {
			var id, orderID string
			var productID, skuID sql.NullString
			var quantity int
			err := rows.Scan(&id, &orderID, &productID, &skuID, &quantity)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to scan row", "details": err.Error()})
				return
			}
			items = append(items, map[string]interface{}{
				"id": id,
				"order_id": orderID,
				"product_id": productID.String,
				"sku_id": skuID.String,
				"quantity": quantity,
			})
		}

		c.JSON(200, gin.H{
			"total_items": count,
			"table_columns": columns,
			"sample_items": items,
		})
	})
	
	// Test data endpoint (for development)
	router.POST("/test-customer", handlers.CreateTestCustomer)
	router.GET("/test-customers", handlers.TestGetCustomers)

	// Initialize handlers
	handlers.InitializeHandlers(db)

	// Admin setup route (no auth required for initial setup)
	router.POST("/setup-admin", handlers.CreateAdminUser)

	// Clean admin routes (WORKING VERSION)
	router.POST("/clean-admin/signup", handlers.CleanAdminSignup)
	router.POST("/clean-admin/login", handlers.CleanAdminLogin)

	// Simple admin routes (no auth required)
	router.POST("/simple-admin/signup", handlers.SimpleAdminSignup)
	router.POST("/simple-admin/login", handlers.SimpleAdminLogin)

	// Admin authentication routes (no auth required)
	router.POST("/admin/signup", handlers.AdminSignup)
	router.POST("/admin/login", handlers.AdminLogin)

	// Admin dashboard route
	router.GET("/admin", handlers.AuthMiddleware(), handlers.AdminMiddleware(), handlers.AdminDashboard)

	// API routes
	api := router.Group("/api/v1")
	{
		// Authentication routes
		auth := api.Group("/auth")
		{
			auth.POST("/send-otp", handlers.SendOTP)
			auth.POST("/verify-otp", handlers.VerifyOTP)
			auth.POST("/refresh", handlers.RefreshToken)
			
			// New phone-based auth routes
			auth.GET("/check-user", handlers.CheckUserExists)
			auth.POST("/login", handlers.LoginUser)
			auth.POST("/register", handlers.RegisterUser)
			auth.POST("/logout", handlers.LogoutUser)
			auth.GET("/validate", handlers.ValidateToken)
			auth.PUT("/update-push-token", handlers.AuthMiddleware(), handlers.UpdatePushToken)
		}

		// Product routes
		products := api.Group("/products")
		{
			products.GET("/", handlers.GetProducts)
			products.GET("/:id", handlers.GetProduct)
			products.GET("/:id/similar", handlers.GetSimilarProducts)
			products.GET("/:id/suggestions", handlers.GetProductSuggestions)
			products.GET("/search", handlers.SearchProductByCode)
			products.GET("/brand/:brandId", handlers.GetProductsByBrand)
			products.POST("/", handlers.CreateProduct)
			products.PUT("/:id", handlers.UpdateProduct)
			products.DELETE("/:id", handlers.DeleteProduct)
			
			// Product view tracking routes
			products.POST("/:id/view", handlers.RegisterProductView)
			products.GET("/most-viewed", handlers.GetMostViewedProducts)
			products.GET("/most-viewed/category/:categoryId", handlers.GetMostViewedProductsByCategory)
			products.GET("/recently-viewed", handlers.AuthMiddleware(), handlers.GetUserRecentlyViewedProducts)
		}

		// Category routes
		categories := api.Group("/categories")
		{
			categories.GET("/", handlers.GetCategories)
			categories.GET("/:id", handlers.GetCategory)
			categories.POST("/", handlers.CreateCategory)
			categories.PUT("/:id", handlers.UpdateCategory)
			categories.DELETE("/:id", handlers.DeleteCategory)
		}

		// Public catalog routes (no auth)
		api.GET("/public/categories", handlers.PublicTopCategories)
		api.GET("/banners", handlers.GetBanners)
		api.GET("/backgrounds", handlers.GetBackgrounds)
		api.GET("/public/products", handlers.SearchProducts)
		api.GET("/public/products/enhanced", handlers.EnhancedSearchProducts)
		api.GET("/public/category-hierarchy", handlers.GetCategoryHierarchy)

		// Brand routes
		brands := api.Group("/brands")
		{
			brands.GET("/", handlers.GetBrands)
			brands.GET("/:id", handlers.GetBrand)
			brands.POST("/", handlers.CreateBrand)
			brands.PUT("/:id", handlers.UpdateBrand)
			brands.DELETE("/:id", handlers.DeleteBrand)
		}

		// User routes (protected)
		users := api.Group("/users")
		users.Use(handlers.AuthMiddleware())
		{
			users.GET("/profile", handlers.GetUserProfile)
			users.PUT("/profile", handlers.UpdateUserProfile)
			users.GET("/orders", handlers.GetUserOrders)
		}

		// Address book routes (protected)
		addresses := api.Group("/addresses")
		addresses.Use(handlers.AuthMiddleware())
		{
			addresses.GET("/", handlers.GetAddressBook)
			addresses.POST("/", handlers.CreateAddress)
			addresses.PUT("/:id", handlers.UpdateAddress)
			addresses.DELETE("/:id", handlers.DeleteAddress)
		}

		// Public location data routes
		api.GET("/cities", handlers.GetCities)
		api.GET("/cities/:cityId/quartiers", handlers.GetQuartiers)

		// Admin location management routes (public for now)
		adminLocations := api.Group("/admin/locations")
		{
			// Cities management
			adminLocations.GET("/cities", handlers.AdminGetCities)
			adminLocations.POST("/cities", handlers.AdminCreateCity)
			adminLocations.PUT("/cities/:id", handlers.AdminUpdateCity)
			adminLocations.DELETE("/cities/:id", handlers.AdminDeleteCity)
			
			// Quartiers management
			adminLocations.GET("/cities/:cityId/quartiers", handlers.AdminGetQuartiers)
			adminLocations.POST("/cities/:cityId/quartiers", handlers.AdminCreateQuartier)
			adminLocations.PUT("/quartiers/:id", handlers.AdminUpdateQuartier)
			adminLocations.DELETE("/quartiers/:id", handlers.AdminDeleteQuartier)
		}

		// Cart routes (protected)
		cart := api.Group("/cart")
		cart.Use(handlers.AuthMiddleware())
		{
			cart.GET("/", handlers.GetCart)
			cart.POST("/add", handlers.AddToCart)
			cart.PUT("/update", handlers.UpdateCartItem)
			cart.DELETE("/remove/:id", handlers.RemoveFromCart)
			cart.DELETE("/clear", handlers.ClearCart)
			cart.POST("/validate", handlers.ValidateCartItems)
		}

		// Order routes (protected)
		orders := api.Group("/orders")
		orders.Use(handlers.AuthMiddleware())
		{
			orders.POST("/", handlers.CreateOrder)
			orders.POST("/upload-payment-proof", handlers.UploadPaymentProof)
			orders.GET("/", handlers.GetUserOrders)
			orders.GET("/:id", handlers.GetOrder)
			// orders.PUT("/:id/cancel", handlers.CancelOrder)
		}

		// Product validation routes (protected)
		productsV := api.Group("/products")
		productsV.Use(handlers.AuthMiddleware())
		{
			products.POST("/:id/validate-variant", handlers.ValidateProductVariant)
		}

		// Wishlist routes (protected)
		wishlist := api.Group("/wishlist")
		wishlist.Use(handlers.AuthMiddleware())
		{
			wishlist.GET("/", handlers.GetWishlist)
			wishlist.POST("/add", handlers.AddToWishlist)
			wishlist.DELETE("/remove/:product_id", handlers.RemoveFromWishlist)
			wishlist.GET("/check/:product_id", handlers.CheckWishlistStatus)
			wishlist.DELETE("/clear", handlers.ClearWishlist)
		}

		// Promotional codes routes
		promo := api.Group("/promotional-codes")
		{
			// Public routes (no auth required)
			promo.POST("/validate", handlers.ValidatePromotionalCode)
			promo.POST("/apply", handlers.ApplyPromotionalCode)
			
			// Admin routes (auth required)
			admin := promo.Group("/admin")
			admin.Use(handlers.AuthMiddleware())
			{
				admin.GET("/", handlers.AdminGetPromotionalCodes)
				admin.POST("/", handlers.AdminCreatePromotionalCode)
				admin.PUT("/:id", handlers.AdminUpdatePromotionalCode)
				admin.DELETE("/:id", handlers.AdminDeletePromotionalCode)
				admin.GET("/stats", handlers.GetPromotionalCodeStats)
			}
		}

		// Admin routes (protected with admin middleware)
		admin := api.Group("/admin")
		admin.Use(handlers.AuthMiddleware(), handlers.AdminMiddleware())
		{
			admin.GET("/stats", handlers.GetAdminStats)
			admin.GET("/users", handlers.GetAllUsers)
			admin.GET("/users/:id", handlers.GetUserByID)
			admin.PUT("/users/:id/role", handlers.UpdateUserRole)
			admin.PUT("/users/:id/status", handlers.ToggleUserStatus)
			admin.PUT("/users/:id/profile", handlers.UpdateUserProfileAdmin)
			admin.GET("/users-stats", handlers.GetUsersStats)
			admin.GET("/orders", handlers.GetAdminOrders)
			admin.GET("/orders/:id", handlers.GetOrderDetails)
			admin.PUT("/orders/:id/status", handlers.UpdateOrderStatus)
			
			// Admin quartier management
			admin.GET("/quartiers", handlers.GetAdminQuartiers)
			admin.PUT("/quartiers/:id/delivery-fee", handlers.UpdateQuartierDeliveryFee)
			admin.GET("/products", handlers.GetAdminProducts)
			admin.GET("/products/:id", handlers.GetAdminProduct)
			admin.GET("/products/:id/skus", handlers.GetProductSKUs)
			admin.POST("/products", handlers.CreateProduct)
			admin.PUT("/products/:id", handlers.UpdateProduct)
			admin.DELETE("/products/:id", handlers.DeleteProduct)
			admin.POST("/upload", handlers.UploadImage)
			
		// Barcode management
		admin.POST("/barcode/scan", handlers.ScanBarcode)
		admin.GET("/barcode/generate/:ean", handlers.GenerateBarcodeImage)
		
		// Public barcode scan (no auth required)
		api.POST("/barcode/scan", handlers.ScanBarcode)
		api.GET("/track/:orderNumber", handlers.TrackOrder)
		
		// Public quartier delivery fees
		api.GET("/quartiers/:id/delivery-fee", handlers.GetQuartierDeliveryFee)
		api.GET("/quartiers/city/:cityId", handlers.GetQuartiersByCity)
		api.GET("/quartiers", handlers.GetAllQuartiers)
			
			// Payment methods management
			admin.GET("/payment-methods", handlers.GetPaymentMethods)
			admin.POST("/payment-methods", handlers.CreatePaymentMethod)
			admin.GET("/payment-methods/:id", handlers.GetPaymentMethod)
			admin.PUT("/payment-methods/:id", handlers.UpdatePaymentMethod)
			admin.DELETE("/payment-methods/:id", handlers.DeletePaymentMethod)
			admin.PUT("/payment-methods/:id/status", handlers.TogglePaymentMethodStatus)
			
		// Banner management
		admin.GET("/banners", handlers.GetAdminBanners)
		admin.POST("/banners", handlers.CreateBanner)
		admin.PUT("/banners/:id", handlers.UpdateBanner)
		admin.POST("/banners/upload-image", handlers.UploadBannerImage)
		admin.GET("/banners/stats", handlers.GetBannerStats)
		
		// Background management1
		admin.GET("/backgrounds", handlers.GetAdminBackgrounds)
		admin.POST("/backgrounds", handlers.CreateBackground)
		admin.PUT("/backgrounds/:id", handlers.UpdateBackground)
		admin.DELETE("/backgrounds/:id", handlers.DeleteBackground)
		admin.POST("/backgrounds/upload-image", handlers.UploadBackgroundImage)
		admin.GET("/backgrounds/stats", handlers.GetBackgroundStats)
		admin.GET("/backgrounds/check-schema", handlers.CheckBackgroundSchema)
		}

		// Admin POS routes (protected with admin or employee)
		adminPOS := api.Group("/admin/pos")
		adminPOS.Use(handlers.AuthMiddleware(), handlers.AdminOrEmployeeMiddleware())
		{
			adminPOS.GET("/orders", handlers.AdminListPOSOrders)
			adminPOS.GET("/orders/:id", handlers.AdminGetPOSOrder)
			adminPOS.GET("/stats", handlers.AdminPOSStats)
		}

		// Inventory admin routes (protected with admin or employee)
		inventory := api.Group("/admin/inventory")
		inventory.Use(handlers.AuthMiddleware(), handlers.AdminOrEmployeeMiddleware())
		{
			inventory.GET("/low-stock", handlers.AdminLowStock)
			inventory.GET("/all", handlers.AdminAllInventory)
			inventory.PUT("/:sku_id/reorder-point", handlers.AdminSetReorderPoint)
			inventory.PUT("/:sku_id/quantity", handlers.AdminUpdateQuantity)
		}

		// CRM routes (protected with admin or employee middleware)
		crm := api.Group("/admin/crm")
		crm.Use(handlers.AuthMiddleware(), handlers.AdminOrEmployeeMiddleware())
		{
			// Customer management
			crm.GET("/customers", handlers.GetCustomers)
			crm.POST("/customers", handlers.CreateCustomer)
			crm.GET("/customers/:id", handlers.GetCustomer)
			crm.GET("/customers/:id/orders", handlers.GetCustomerOrders)
			crm.PUT("/customers/:id", handlers.UpdateCustomer)
			crm.DELETE("/customers/:id", handlers.DeleteCustomer)
			
			// Customer interactions (using different path structure)
			crm.GET("/customers/:id/interactions", handlers.GetCustomerInteractions)
			crm.POST("/customers/:id/interactions", handlers.CreateCustomerInteraction)
			crm.PUT("/interactions/:id", handlers.UpdateCustomerInteraction)
			crm.DELETE("/interactions/:id", handlers.DeleteCustomerInteraction)
			
			// Customer statistics
			crm.GET("/customers/:id/stats", handlers.GetCustomerStats)
			crm.GET("/stats", handlers.GetCRMStats)
		}

		// POS routes (protected with admin or employee middleware)
		pos := api.Group("/pos")
		pos.Use(handlers.AuthMiddleware(), handlers.AdminOrEmployeeMiddleware())
		{
			pos.GET("/catalog", handlers.GetPOSCatalog)
			pos.GET("/customers", handlers.GetPOSCustomers)
			pos.GET("/product-models/:product_model_id/variants", handlers.GetProductVariants)
			pos.GET("/payment-methods", handlers.GetActivePaymentMethods)
			pos.POST("/orders", handlers.CreatePOSOrder)
		}
	}

	// Start server
	log.Printf("Starting FMBQ Server on 0.0.0.0:%s", config.AppConfig.ServerPort)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+config.AppConfig.ServerPort, c.Handler(router)))
}
