package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"fmbq-server/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetSuggestedForYou handles GET /api/v1/products/suggested-for-you
// Returns personalized product suggestions based on view history
// Priority: Category (level 1, 2, 3) then Brand
func GetSuggestedForYou(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 20
	}

	// Get user ID or phone number or anonymous session ID
	var userID uuid.NullUUID
	var phoneNumber string
	
	// Try to get authenticated user ID from context (if AuthMiddleware was used)
	if userIDInterface, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userIDInterface.(string); ok {
			if parsedUserID, err := uuid.Parse(userIDStr); err == nil {
				userID = uuid.NullUUID{UUID: parsedUserID, Valid: true}
				fmt.Printf("📊 Got user_id from context: %s\n", userID.UUID.String())
			}
		}
		// Get phone from context (can be "phone" or "user_phone")
		if phoneInterface, exists := c.Get("phone"); exists {
			if phoneStr, ok := phoneInterface.(string); ok {
				phoneNumber = phoneStr
			}
		}
		if phoneNumber == "" {
			if phoneInterface, exists := c.Get("user_phone"); exists {
				if phoneStr, ok := phoneInterface.(string); ok {
					phoneNumber = phoneStr
				}
			}
		}
	}
	
	// If not in context, try to validate token from header (optional auth)
	if !userID.Valid {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString := strings.TrimSpace(authHeader[7:])
			if tokenString != "" {
				// Validate token from database
				var fetchedUserID uuid.UUID
				var fetchedPhone sql.NullString
				query := `SELECT ut.user_id, u.phone 
				          FROM user_tokens ut
				          JOIN users u ON ut.user_id = u.id
				          WHERE ut.token = $1 AND ut.revoked = false AND u.is_active = true`
				err := database.Database.QueryRow(query, tokenString).Scan(&fetchedUserID, &fetchedPhone)
				if err == nil {
					userID = uuid.NullUUID{UUID: fetchedUserID, Valid: true}
					if fetchedPhone.Valid {
						phoneNumber = fetchedPhone.String
					}
					fmt.Printf("✅ Got user_id from token validation: %s\n", userID.UUID.String())
				}
			}
		}
	}
	
	// If still not authenticated, try to get user_id from query (for better tracking)
	if !userID.Valid {
		userIDStr := c.Query("user_id")
		if userIDStr != "" {
			if parsedUserID, err := uuid.Parse(userIDStr); err == nil {
				userID = uuid.NullUUID{UUID: parsedUserID, Valid: true}
				fmt.Printf("📱 Found user_id in query: %s\n", userIDStr)
			}
		}
	}
	
	// If still not authenticated, get phone from query
	if !userID.Valid {
		phoneNumber = c.Query("phone_number")
	}

	// Get user's viewed products (last 30 days, ordered by last viewed time)
	var viewedProductIDs []uuid.UUID
	var queryArgs []interface{}
	var queryCondition string

	// Determine which identifier to use (priority: user_id > phone_number > anonymous_session_id)
	if userID.Valid {
		queryCondition = `WHERE pv.user_id = $1 AND pv.view_timestamp > NOW() - INTERVAL '30 days'`
		queryArgs = append(queryArgs, userID.UUID)
		fmt.Printf("📊 Using user_id for suggestions: %s\n", userID.UUID.String())
	} else if phoneNumber != "" {
		queryCondition = `WHERE pv.phone_number = $1 AND pv.view_timestamp > NOW() - INTERVAL '30 days'`
		queryArgs = append(queryArgs, phoneNumber)
		fmt.Printf("📊 Using phone_number for suggestions: %s\n", phoneNumber)
	} else {
		// Try to get anonymous session ID from header or query
		anonymousSessionID := c.GetHeader("X-Anonymous-Session-ID")
		if anonymousSessionID == "" {
			anonymousSessionID = c.Query("anonymous_session_id")
		}

		if anonymousSessionID == "" {
			fmt.Printf("⚠️ No user identifier found (user_id, phone_number, or anonymous_session_id)\n")
			// Return empty suggestions if no identifier
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    []interface{}{},
				"count":   0,
				"message": "No user identifier provided",
			})
			return
		}

		// Use anonymous session ID to find views
		queryCondition = `WHERE pv.anonymous_session_id = $1 AND pv.view_timestamp > NOW() - INTERVAL '30 days'`
		queryArgs = append(queryArgs, anonymousSessionID)
		fmt.Printf("📊 Using anonymous_session_id for suggestions: %s\n", anonymousSessionID[:20]+"...")
	}

	// Fix: For SELECT DISTINCT + ORDER BY, all ORDER BY columns must also be in SELECT list.
	// So we select both product_id and MAX(pv.view_timestamp), then pick only product_id in Go.
	viewedQuery := fmt.Sprintf(`
		SELECT pv.product_id, MAX(pv.view_timestamp) as last_viewed
		FROM product_views pv
		%s
		GROUP BY pv.product_id
		ORDER BY last_viewed DESC
		LIMIT 50
	`, queryCondition)

	rows, err := database.Database.Query(viewedQuery, queryArgs...)
	if err != nil {
		fmt.Printf("❌ Error fetching viewed products: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch view history"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var productID uuid.UUID
		var lastViewed sql.NullTime // Not strictly needed, but required since we select it.
		if err := rows.Scan(&productID, &lastViewed); err == nil {
			viewedProductIDs = append(viewedProductIDs, productID)
		}
	}

	fmt.Printf("📊 Found %d viewed products for user (user_id: %v, phone: %s)\n",
		len(viewedProductIDs), userID.Valid, phoneNumber)

	if len(viewedProductIDs) == 0 {
		fmt.Printf("⚠️ No viewed products found, returning empty suggestions\n")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    []interface{}{},
			"count":   0,
			"message": "No view history found",
		})
		return
	}

	// Build query to find suggested products with priority
	// Priority order: Category level 1 > Category level 2 > Category level 3 > Brand
	suggestedQuery := `
		WITH viewed_product_ids AS (
			SELECT UNNEST($1::uuid[]) as product_id
		),
		viewed_product_data AS (
			SELECT DISTINCT
				vp.product_id,
				pm.brand_id,
				array_agg(DISTINCT c.id) FILTER (WHERE c.level = 1) as level1_categories,
				array_agg(DISTINCT c.id) FILTER (WHERE c.level = 2) as level2_categories,
				array_agg(DISTINCT c.id) FILTER (WHERE c.level = 3) as level3_categories
			FROM viewed_product_ids vp
			JOIN product_models pm ON vp.product_id = pm.id
			LEFT JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
			LEFT JOIN categories c ON pmc.category_id = c.id
			GROUP BY vp.product_id, pm.brand_id
		),
		candidate_products_raw AS (
			SELECT DISTINCT
				pm.id,
				pm.title,
				pm.model_code,
				pm.description,
				b.name as brand_name,
				b.id as brand_id,
				b.color as brand_color,
				-- Priority scoring (most specific first):
				-- 4 = same level3 category, 3 = same level2, 2 = same level1, 1 = same brand, 0 = none
				CASE 
					WHEN EXISTS (
						SELECT 1 FROM viewed_product_data vpd
						JOIN product_model_categories pmc2 ON pmc2.product_model_id = pm.id
						JOIN categories c2 ON pmc2.category_id = c2.id
						WHERE c2.id = ANY(vpd.level3_categories) AND c2.level = 3
					) THEN 4
					WHEN EXISTS (
						SELECT 1 FROM viewed_product_data vpd
						JOIN product_model_categories pmc2 ON pmc2.product_model_id = pm.id
						JOIN categories c2 ON pmc2.category_id = c2.id
						WHERE c2.id = ANY(vpd.level2_categories) AND c2.level = 2
					) THEN 3
					WHEN EXISTS (
						SELECT 1 FROM viewed_product_data vpd
						JOIN product_model_categories pmc2 ON pmc2.product_model_id = pm.id
						JOIN categories c2 ON pmc2.category_id = c2.id
						WHERE c2.id = ANY(vpd.level1_categories) AND c2.level = 1
					) THEN 2
					WHEN EXISTS (
						SELECT 1 FROM viewed_product_data vpd
						WHERE vpd.brand_id = pm.brand_id AND pm.brand_id IS NOT NULL
					) THEN 1
					ELSE 0
				END as priority_score,
				COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) as min_price,
				COALESCE(MAX(pr.list_price), 0) as original_price
			FROM product_models pm
			LEFT JOIN brands b ON pm.brand_id = b.id
			LEFT JOIN skus s ON pm.id = s.product_model_id
			LEFT JOIN prices pr ON s.id = pr.sku_id AND pr.currency = 'MRO'
			CROSS JOIN viewed_product_data vpd
			WHERE pm.is_active = true
			AND pm.id != ALL($1::uuid[])
			GROUP BY pm.id, pm.title, pm.model_code, pm.description, b.name, b.id, b.color, vpd.level1_categories, vpd.level2_categories, vpd.level3_categories, vpd.brand_id
		),
		candidate_products AS (
			SELECT *
			FROM candidate_products_raw
			WHERE priority_score > 0
		),
		products_with_images AS (
			SELECT 
				cp.*,
				COALESCE(pi.url, '') as main_image_url
			FROM candidate_products cp
			LEFT JOIN LATERAL (
				SELECT url 
				FROM product_images 
				WHERE product_model_id = cp.id 
				ORDER BY position, created_at
				LIMIT 1
			) pi ON true
			WHERE pi.url IS NOT NULL AND pi.url != ''
		)
		SELECT * FROM products_with_images
		ORDER BY priority_score DESC, min_price ASC
		LIMIT $2
	`

	// Convert viewedProductIDs to PostgreSQL array format
	productIDsArray := fmt.Sprintf("{%s}", func() string {
		var ids []string
		for _, id := range viewedProductIDs {
			ids = append(ids, fmt.Sprintf(`"%s"`, id.String()))
		}
		return strings.Join(ids, ",")
	}())

	fmt.Printf("🔍 Querying suggested products with %d viewed product IDs\n", len(viewedProductIDs))
	suggestedRows, err := database.Database.Query(suggestedQuery, productIDsArray, limit)
	if err != nil {
		fmt.Printf("❌ Error fetching suggested products: %v\n", err)
		fmt.Printf("Query: %s\n", suggestedQuery)
		fmt.Printf("Args: productIDsArray=%s, limit=%d\n", productIDsArray, limit)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch suggested products: %v", err)})
		return
	}
	defer suggestedRows.Close()

	var products []map[string]interface{}
	for suggestedRows.Next() {
		var (
			productID          uuid.UUID
			title, modelCode   sql.NullString
			description        sql.NullString
			brandName          sql.NullString
			brandID            uuid.NullUUID
			brandColor         sql.NullString
			priorityScore      int
			minPrice, originalPrice sql.NullFloat64
			mainImageURL       sql.NullString
		)

		err := suggestedRows.Scan(
			&productID, &title, &modelCode, &description,
			&brandName, &brandID, &brandColor, &priorityScore,
			&minPrice, &originalPrice, &mainImageURL,
		)
		if err != nil {
			fmt.Printf("Error scanning suggested product: %v\n", err)
			continue
		}

		product := map[string]interface{}{
			"id":             productID.String(),
			"name":           title.String,
			"title":          title.String,
			"model_code":     modelCode.String,
			"description":    description.String,
			"brand_name":     brandName.String,
			"brand_id":       brandID.UUID.String(),
			"brand_color":    brandColor.String,
			"image_url":      mainImageURL.String,
			"price":          minPrice.Float64,
			"original_price": originalPrice.Float64,
			"priority_score": priorityScore,
			"is_favorite":    false,
		}

		products = append(products, product)
	}

	fmt.Printf("✅ Found %d suggested products\n", len(products))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    products,
		"count":   len(products),
	})
}
