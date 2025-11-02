package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

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

// GetUserRecentlyViewedProducts handles GET /api/v1/products/recently-viewed
// Returns products the authenticated user has recently viewed, ordered by most recent first
func GetUserRecentlyViewedProducts(c *gin.Context) {
	// Get user ID from auth context (required for this endpoint)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userIDStr, ok := userIDInterface.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 10
	}

	// Get user's recently viewed products, ordered by most recent first
	// First get distinct product IDs with their latest view timestamp
	query := `
		WITH latest_views AS (
			SELECT DISTINCT ON (pv.product_id)
				pv.product_id,
				pv.view_timestamp
			FROM product_views pv
			WHERE pv.user_id = $1
			ORDER BY pv.product_id, pv.view_timestamp DESC
		)
		SELECT 
			lv.product_id,
			lv.view_timestamp,
			pm.id,
			pm.title,
			pm.model_code,
			pm.description,
			b.name as brand_name,
			b.color as brand_color,
			COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) as min_price,
			COALESCE(MAX(pr.list_price), 0) as original_price,
			COALESCE(pi.url, '') as main_image_url
		FROM latest_views lv
		INNER JOIN product_models pm ON lv.product_id = pm.id
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN prices pr ON s.id = pr.sku_id AND pr.currency = 'MRO'
		LEFT JOIN LATERAL (
			SELECT url 
			FROM product_images 
			WHERE product_model_id = pm.id 
			ORDER BY position, created_at
			LIMIT 1
		) pi ON true
		WHERE pm.is_active = true
		GROUP BY lv.product_id, lv.view_timestamp, pm.id, pm.title, pm.model_code, pm.description, b.name, b.color, pi.url
		ORDER BY lv.view_timestamp DESC
		LIMIT $2
	`

	rows, err := database.Database.Query(query, userID, limit)
	if err != nil {
		fmt.Printf("Error fetching recently viewed products: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recently viewed products"})
		return
	}
	defer rows.Close()

	type productWithTime struct {
		product     map[string]interface{}
		viewTime    time.Time
	}

	var productsWithTime []productWithTime

	for rows.Next() {
		var (
			productID          uuid.UUID
			viewTimestamp     time.Time
			pmID               uuid.UUID
			title, modelCode  sql.NullString
			description       sql.NullString
			brandName         sql.NullString
			brandColor        sql.NullString
			minPrice, originalPrice sql.NullFloat64
			mainImageURL      sql.NullString
		)

		err := rows.Scan(
			&productID, &viewTimestamp, &pmID, &title, &modelCode, &description,
			&brandName, &brandColor, &minPrice, &originalPrice, &mainImageURL,
		)
		if err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}

		// Fetch all images for this product
		imagesQuery := `SELECT id, url, alt, position, created_at 
		                FROM product_images WHERE product_model_id = $1 ORDER BY position, created_at`
		imagesRows, err := database.Database.Query(imagesQuery, pmID)
		var images []map[string]interface{}
		if err == nil {
			defer imagesRows.Close()
			for imagesRows.Next() {
				var imgID, imgURL, imgAlt sql.NullString
				var imgPosition sql.NullInt32
				var imgCreatedAt time.Time
				if err := imagesRows.Scan(&imgID, &imgURL, &imgAlt, &imgPosition, &imgCreatedAt); err == nil {
					images = append(images, map[string]interface{}{
						"id":         imgID.String,
						"url":        imgURL.String,
						"alt":        imgAlt.String,
						"position":   imgPosition.Int32,
						"created_at": imgCreatedAt,
					})
				}
			}
		}

		productData := map[string]interface{}{
			"id":             pmID.String(),
			"title":          title.String,
			"model_code":     modelCode.String,
			"description":    description.String,
			"brand_name":     brandName.String,
			"brand_color":    brandColor.String,
			"image_url":      mainImageURL.String,
			"images":         images,
			"price":          minPrice.Float64,
			"original_price": originalPrice.Float64,
		}

		productsWithTime = append(productsWithTime, productWithTime{
			product:  productData,
			viewTime: viewTimestamp,
		})
	}

	// Sort by view_timestamp (most recent first)
	sort.Slice(productsWithTime, func(i, j int) bool {
		return productsWithTime[i].viewTime.After(productsWithTime[j].viewTime)
	})

	// Remove duplicates (keep first occurrence which is most recent)
	seen := make(map[string]bool)
	var uniqueProducts []map[string]interface{}
	for _, pwt := range productsWithTime {
		id := pwt.product["id"].(string)
		if !seen[id] {
			seen[id] = true
			uniqueProducts = append(uniqueProducts, pwt.product)
			if len(uniqueProducts) >= limit {
				break
			}
		}
	}

	fmt.Printf("✅ Found %d recently viewed products for user %s\n", len(uniqueProducts), userID.String())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    uniqueProducts,
		"count":   len(uniqueProducts),
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
