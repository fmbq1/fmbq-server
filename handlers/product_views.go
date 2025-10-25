package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"fmbq-server/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RegisterProductView handles POST /api/v1/products/:id/view
func RegisterProductView(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Verify product exists
	var productExists bool
	err = database.Database.QueryRow("SELECT EXISTS(SELECT 1 FROM product_models WHERE id = $1)", productID).Scan(&productExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify product"})
		return
	}
	if !productExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Get user ID from auth context (if authenticated)
	var userID uuid.NullUUID
	if userIDInterface, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userIDInterface.(string); ok {
			if parsedUserID, err := uuid.Parse(userIDStr); err == nil {
				userID = uuid.NullUUID{UUID: parsedUserID, Valid: true}
			}
		}
	}

	// Generate anonymous session ID if not authenticated
	var anonymousSessionID string
	if !userID.Valid {
		anonymousSessionID = uuid.New().String()
	}

	// Get client info
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// Check for recent duplicate view (within 5 minutes)
	var duplicateExists bool
	if userID.Valid {
		err = database.Database.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM product_views 
				WHERE product_id = $1 AND user_id = $2 
				AND view_timestamp > NOW() - INTERVAL '5 minutes'
			)`, productID, userID.UUID).Scan(&duplicateExists)
	} else {
		err = database.Database.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM product_views 
				WHERE product_id = $1 AND anonymous_session_id = $2 
				AND view_timestamp > NOW() - INTERVAL '5 minutes'
			)`, productID, anonymousSessionID).Scan(&duplicateExists)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for duplicate views"})
		return
	}

	if duplicateExists {
		c.JSON(http.StatusOK, gin.H{"message": "View already registered recently", "success": true})
		return
	}

	// Insert the view
	viewID := uuid.New()
	_, err = database.Database.Exec(`
		INSERT INTO product_views (id, product_id, user_id, anonymous_session_id, view_timestamp, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6)
	`, viewID, productID, userID, anonymousSessionID, ipAddress, userAgent)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register view"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "View registered successfully", "success": true})
}

// GetMostViewedProducts handles GET /api/v1/products/most-viewed
func GetMostViewedProducts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 10
	}

	categoryIDStr := c.Query("category_id")
	var categoryFilter string
	var args []interface{}
	
	if categoryIDStr != "" {
		categoryID, err := uuid.Parse(categoryIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}
		categoryFilter = `
			AND EXISTS (
				SELECT 1 FROM product_model_categories pmc 
				JOIN categories c ON pmc.category_id = c.id 
				WHERE pmc.product_model_id = pm.id 
				AND c.parent_id = $2
			)`
		args = []interface{}{limit, categoryID}
	} else {
		args = []interface{}{limit}
	}

	// Simplified query for debugging
	query := fmt.Sprintf(`
		SELECT 
			pm.id,
			pm.title,
			pm.model_code,
			pm.brand_id,
			b.name as brand_name,
			COALESCE(view_counts.view_count, 0) as view_count,
			(SELECT pi.url FROM product_images pi WHERE pi.product_model_id = pm.id ORDER BY pi.created_at LIMIT 1) as main_image_url,
			(SELECT pi.url FROM product_images pi 
			 JOIN product_colors pc ON pi.product_color_id = pc.id 
			 WHERE pc.product_model_id = pm.id AND pi.url IS NOT NULL 
			 ORDER BY pi.created_at LIMIT 1) as color_image_url,
			(SELECT p.sale_price FROM prices p 
			 JOIN skus s ON p.sku_id = s.id 
			 WHERE s.product_model_id = pm.id AND p.sale_price IS NOT NULL 
			 ORDER BY p.created_at DESC LIMIT 1) as price,
			(SELECT p.list_price FROM prices p 
			 JOIN skus s ON p.sku_id = s.id 
			 WHERE s.product_model_id = pm.id 
			 ORDER BY p.created_at DESC LIMIT 1) as original_price
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN (
			SELECT 
				product_id,
				COUNT(*) as view_count
			FROM product_views
			WHERE view_timestamp > NOW() - INTERVAL '30 days'
			GROUP BY product_id
		) view_counts ON pm.id = view_counts.product_id
		WHERE pm.id IS NOT NULL
		%s
		ORDER BY view_counts.view_count DESC NULLS LAST, pm.created_at DESC
		LIMIT $1
	`, categoryFilter)

	rows, err := database.Database.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch most viewed products: %v", err)})
		return
	}
	defer rows.Close()

	var products []map[string]interface{}
	for rows.Next() {
		var (
			id, title, modelCode                                                                 sql.NullString
			brandID                                                                             sql.NullString
			brandName                                                                           sql.NullString
			viewCount                                                                           sql.NullInt64
			mainImageURL, colorImageURL                                                         sql.NullString
			price, originalPrice                                                                sql.NullFloat64
		)

		err := rows.Scan(
			&id, &title, &modelCode, &brandID, &brandName, &viewCount, &mainImageURL, &colorImageURL, &price, &originalPrice,
		)
		if err != nil {
			continue
		}

		// Determine the best image to use
		var imageURL string
		if mainImageURL.Valid && mainImageURL.String != "" {
			imageURL = mainImageURL.String
		} else if colorImageURL.Valid && colorImageURL.String != "" {
			imageURL = colorImageURL.String
		}

		// Skip products without images
		if imageURL == "" {
			continue
		}

		product := map[string]interface{}{
			"id":             id.String,
			"name":           title.String,
			"model_code":     modelCode.String,
			"brand_id":       brandID.String,
			"brand_name":     brandName.String,
			"view_count":     viewCount.Int64,
			"image_url":      imageURL,
			"price":          price.Float64,
			"original_price": originalPrice.Float64,
			"is_favorite":    false,
		}

		products = append(products, product)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    products,
		"count":   len(products),
	})
}

// GetMostViewedProductsByCategory handles GET /api/v1/products/most-viewed/category/:categoryId
func GetMostViewedProductsByCategory(c *gin.Context) {
	categoryIDStr := c.Param("categoryId")
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 10
	}

	query := `
		SELECT 
			pm.id,
			pm.title,
			pm.model_code,
			pm.brand_id,
			b.name as brand_name,
			COALESCE(view_counts.view_count, 0) as view_count,
			-- Get main product image
			(SELECT pi.url FROM product_images pi WHERE pi.product_model_id = pm.id ORDER BY pi.created_at LIMIT 1) as main_image_url,
			-- Get first color image as fallback
			(SELECT pi.url FROM product_images pi 
			 JOIN product_colors pc ON pi.product_color_id = pc.id 
			 WHERE pc.product_model_id = pm.id AND pi.url IS NOT NULL 
			 ORDER BY pi.created_at LIMIT 1) as color_image_url,
			-- Get price info
			(SELECT p.sale_price FROM prices p 
			 JOIN skus s ON p.sku_id = s.id 
			 WHERE s.product_model_id = pm.id AND p.sale_price IS NOT NULL 
			 ORDER BY p.created_at DESC LIMIT 1) as price,
			(SELECT p.list_price FROM prices p 
			 JOIN skus s ON p.sku_id = s.id 
			 WHERE s.product_model_id = pm.id 
			 ORDER BY p.created_at DESC LIMIT 1) as original_price
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN (
			SELECT 
				product_id,
				COUNT(*) as view_count
			FROM product_views
			WHERE view_timestamp > NOW() - INTERVAL '30 days'
			GROUP BY product_id
		) view_counts ON pm.id = view_counts.product_id
		WHERE EXISTS (
			SELECT 1 FROM product_model_categories pmc 
			JOIN categories c ON pmc.category_id = c.id 
			WHERE pmc.product_model_id = pm.id 
			AND c.parent_id = $1
		)
		AND (
			(SELECT pi.url FROM product_images pi WHERE pi.product_model_id = pm.id ORDER BY pi.created_at LIMIT 1) IS NOT NULL
			OR (SELECT pi.url FROM product_images pi 
				JOIN product_colors pc ON pi.product_color_id = pc.id 
				WHERE pc.product_model_id = pm.id AND pi.url IS NOT NULL 
				ORDER BY pi.created_at LIMIT 1) IS NOT NULL
		)
		ORDER BY view_counts.view_count DESC NULLS LAST, pm.created_at DESC
		LIMIT $2
	`

	rows, err := database.Database.Query(query, categoryID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch most viewed products by category"})
		return
	}
	defer rows.Close()

	var products []map[string]interface{}
	for rows.Next() {
		var (
			id, title, modelCode sql.NullString
			brandID              sql.NullString
			brandName            sql.NullString
			viewCount            sql.NullInt64
			mainImageURL, colorImageURL sql.NullString
			price, originalPrice sql.NullFloat64
		)

		err := rows.Scan(
			&id, &title, &modelCode, &brandID, &brandName, &viewCount, 
			&mainImageURL, &colorImageURL, &price, &originalPrice,
		)
		if err != nil {
			continue
		}

		// Determine the best image to use
		var imageURL string
		if mainImageURL.Valid && mainImageURL.String != "" {
			imageURL = mainImageURL.String
		} else if colorImageURL.Valid && colorImageURL.String != "" {
			imageURL = colorImageURL.String
		}

		// Skip products without images
		if imageURL == "" {
			continue
		}

		product := map[string]interface{}{
			"id":             id.String,
			"name":           title.String,
			"model_code":     modelCode.String,
			"brand_id":       brandID.String,
			"brand_name":     brandName.String,
			"view_count":     viewCount.Int64,
			"image_url":      imageURL,
			"price":          price.Float64,
			"original_price": originalPrice.Float64,
			"is_favorite":    false, // Will be set by frontend if user is authenticated
		}

		products = append(products, product)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    products,
		"count":   len(products),
	})
}
