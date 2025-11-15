package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"fmbq-server/database"
	"fmbq-server/models"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetWishlist retrieves user's wishlist with product details
func GetWishlist(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse pagination parameters
	page := 1
	limit := 20
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Query wishlist items with complete product details including images, colors, and prices
	query := `
		SELECT 
			wi.id,
			wi.user_id,
			wi.product_id,
			wi.created_at,
			wi.updated_at,
			p.id as product_id,
			p.title,
			p.description,
			p.brand_id,
			b.name as brand_name,
			b.color as brand_color,
			p.model_code,
			p.created_at as product_created_at,
			p.updated_at as product_updated_at,
			-- Get the first product image
			COALESCE(
				(SELECT pi.url 
				 FROM product_images pi 
				 WHERE pi.product_model_id = p.id 
				 ORDER BY pi.created_at ASC 
				 LIMIT 1),
				''
			) as main_image_url
		FROM wishlist_items wi
		JOIN product_models p ON wi.product_id = p.id
		JOIN brands b ON p.brand_id = b.id
		WHERE wi.user_id = $1
		ORDER BY wi.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := database.Database.Query(query, userID, limit, offset)
	if err != nil {
		fmt.Printf("❌ Wishlist query error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch wishlist"})
		return
	}
	defer rows.Close()

	var wishlistItems []models.WishlistItemWithProduct
	for rows.Next() {
		var item models.WishlistItemWithProduct
		var productID, productTitle, productDescription, brandID, brandName, brandColor, modelCode, mainImageURL string
		var productCreatedAt, productUpdatedAt time.Time

		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.ProductID,
			&item.CreatedAt,
			&item.UpdatedAt,
			&productID,
			&productTitle,
			&productDescription,
			&brandID,
			&brandName,
			&brandColor,
			&modelCode,
			&productCreatedAt,
			&productUpdatedAt,
			&mainImageURL,
		)
		if err != nil {
			fmt.Printf("❌ Wishlist scan error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan wishlist item"})
			return
		}

		// Get product colors with their images
		colorsQuery := `
			SELECT 
				pc.id, 
				pc.color_name, 
				pc.is_active,
				COALESCE(
					(SELECT pi.url 
					 FROM product_images pi 
					 WHERE pi.color_id = pc.id 
					 ORDER BY pi.created_at ASC 
					 LIMIT 1),
					''
				) as color_image_url
			FROM product_colors pc
			WHERE pc.product_model_id = $1 AND pc.is_active = true
			ORDER BY pc.created_at ASC
		`
		colorRows, err := database.Database.Query(colorsQuery, productID)
		var colors []map[string]interface{}
		if err != nil {
			fmt.Printf("❌ Colors query error for product %s: %v\n", productID, err)
		} else {
			for colorRows.Next() {
				var colorID, colorName, colorImageURL string
				var isActive bool
				if err := colorRows.Scan(&colorID, &colorName, &isActive, &colorImageURL); err != nil {
					fmt.Printf("❌ Color scan error: %v\n", err)
				} else {
					colors = append(colors, map[string]interface{}{
						"id":         colorID,
						"color_name": colorName,
						"color_hex":  "", // No color_hex column exists
						"image_url":  colorImageURL, // Now getting from product_images
						"is_active":  isActive,
					})
				}
			}
			colorRows.Close()
		}

		// Get product prices (simplified query without color_id reference)
		var minPrice, maxPrice float64
		priceQuery := `
			SELECT MIN(price), MAX(price)
			FROM skus s
			WHERE s.product_model_id = $1 AND s.is_active = true
		`
		err = database.Database.QueryRow(priceQuery, productID).Scan(&minPrice, &maxPrice)
		if err != nil {
			fmt.Printf("❌ Price query error for product %s: %v\n", productID, err)
			minPrice = 0
			maxPrice = 0
		}

		// Determine original price
		var originalPrice float64
		if maxPrice > minPrice {
			originalPrice = maxPrice
		} else {
			originalPrice = minPrice
		}

		// Create product map with complete information
		product := map[string]interface{}{
			"id":             productID,
			"title":          productTitle,
			"description":    productDescription,
			"brand_id":       brandID,
			"brand_name":     brandName,
			"brand_color":    brandColor,
			"model_code":     modelCode,
			"image_url":      mainImageURL, // Use the first color image
			"colors":         colors,
			"price":          minPrice,
			"original_price": originalPrice,
			"is_active":      true,
			"created_at":     productCreatedAt,
			"updated_at":     productUpdatedAt,
		}
		item.Product = product
		wishlistItems = append(wishlistItems, item)
	}

	// Get total count for pagination
	var totalCount int
	countQuery := `
		SELECT COUNT(*)
		FROM wishlist_items wi
		JOIN product_models p ON wi.product_id = p.id
		WHERE wi.user_id = $1
	`
	err = database.Database.QueryRow(countQuery, userID).Scan(&totalCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count wishlist items"})
		return
	}

	// Calculate pagination info
	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"wishlist_items": wishlistItems,
		"pagination": gin.H{
			"current_page": page,
			"total_pages":  totalPages,
			"total_items":  totalCount,
			"limit":       limit,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	})
}

// AddToWishlist adds a product to user's wishlist
func AddToWishlist(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request struct {
		ProductID string `json:"product_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate product exists and is active
	var productExists bool
	err := database.Database.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM product_models WHERE id = $1)",
		request.ProductID,
	).Scan(&productExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate product"})
		return
	}

	if !productExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found or inactive"})
		return
	}

	// Check if already in wishlist
	var alreadyExists bool
	err = database.Database.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM wishlist_items WHERE user_id = $1 AND product_id = $2)",
		userID,
		request.ProductID,
	).Scan(&alreadyExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check wishlist"})
		return
	}

	if alreadyExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Product already in wishlist"})
		return
	}

	// Fetch product metadata (name, image, price) for notifications
	var productName, productImageURL sql.NullString
	var productPrice sql.NullFloat64
	
	metadataQuery := `
		SELECT 
			pm.title,
			COALESCE(
				(SELECT pi.url FROM product_images pi 
				 WHERE pi.product_model_id = pm.id 
				 ORDER BY pi.position ASC LIMIT 1),
				''
			) as image_url,
			COALESCE(
				(SELECT p.sale_price FROM prices p 
				 JOIN skus s ON p.sku_id = s.id 
				 WHERE s.product_model_id = pm.id AND p.currency = 'MRO' 
				 ORDER BY p.sale_price ASC LIMIT 1),
				(SELECT p.list_price FROM prices p 
				 JOIN skus s ON p.sku_id = s.id 
				 WHERE s.product_model_id = pm.id AND p.currency = 'MRO' 
				 ORDER BY p.list_price ASC LIMIT 1),
				0
			) as price
		FROM product_models pm
		WHERE pm.id = $1
	`
	err = database.Database.QueryRow(metadataQuery, request.ProductID).Scan(&productName, &productImageURL, &productPrice)
	if err != nil {
		// If metadata fetch fails, use defaults but continue
		productName = sql.NullString{String: "Product", Valid: true}
		productImageURL = sql.NullString{String: "", Valid: true}
		productPrice = sql.NullFloat64{Float64: 0, Valid: true}
	}

	// Add to wishlist
	wishlistID := uuid.New().String()
	now := time.Now()

	_, err = database.Database.Exec(
		"INSERT INTO wishlist_items (id, user_id, product_id, product_name, product_image_url, product_price, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		wishlistID,
		userID,
		request.ProductID,
		productName.String,
		productImageURL.String,
		productPrice.Float64,
		now,
		now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to wishlist"})
		return
	}

	// Schedule wishlist reminder notifications
	go func() {
		scheduler := services.NewNotificationScheduler()
		productUUID, err := uuid.Parse(request.ProductID)
		if err == nil {
			if err := scheduler.ScheduleWishlistReminders(
				uuid.MustParse(userID),
				productUUID,
				productName.String,
				productImageURL.String,
				productPrice.Float64,
			); err != nil {
				fmt.Printf("⚠️ Failed to schedule wishlist reminders: %v\n", err)
			}
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product added to wishlist",
		"wishlist_item_id": wishlistID,
	})
}

// RemoveFromWishlist removes a product from user's wishlist
func RemoveFromWishlist(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	// Cancel wishlist reminder notifications before removing
	go func() {
		scheduler := services.NewNotificationScheduler()
		productUUID, err := uuid.Parse(productID)
		if err == nil {
			scheduler.CancelWishlistReminders(uuid.MustParse(userID), productUUID)
		}
	}()

	// Remove from wishlist
	result, err := database.Database.Exec(
		"DELETE FROM wishlist_items WHERE user_id = $1 AND product_id = $2",
		userID,
		productID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove from wishlist"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check removal"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found in wishlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product removed from wishlist"})
}

// CheckWishlistStatus checks if a product is in user's wishlist
func CheckWishlistStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	var isInWishlist bool
	err := database.Database.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM wishlist_items WHERE user_id = $1 AND product_id = $2)",
		userID,
		productID,
	).Scan(&isInWishlist)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check wishlist status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_in_wishlist": isInWishlist,
	})
}

// ClearWishlist removes all items from user's wishlist
func ClearWishlist(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	_, err := database.Database.Exec("DELETE FROM wishlist_items WHERE user_id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear wishlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wishlist cleared"})
}
