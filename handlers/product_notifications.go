package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"fmbq-server/database"
	"fmbq-server/models"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscribeToProductNotification allows a user to subscribe to notifications for a product
func SubscribeToProductNotification(c *gin.Context) {
	// Get user ID from context (set by AuthMiddleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		fmt.Println("❌ SubscribeToProductNotification: user_id not in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse user ID - AuthMiddleware stores it as string
	var userID uuid.UUID
	var parseErr error
	if userIDStr, ok := userIDInterface.(string); ok {
		userID, parseErr = uuid.Parse(userIDStr)
	} else if userIDParsed, ok := userIDInterface.(uuid.UUID); ok {
		userID = userIDParsed
	} else {
		fmt.Printf("❌ SubscribeToProductNotification: invalid user_id type: %T, value: %v\n", userIDInterface, userIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}
	
	if parseErr != nil {
		fmt.Printf("❌ SubscribeToProductNotification: failed to parse user_id: %v\n", parseErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		fmt.Printf("❌ SubscribeToProductNotification: invalid product ID: %s, error: %v\n", productIDStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Optional SKU ID from query parameter
	var skuID *uuid.UUID
	if skuIDStr := c.Query("sku_id"); skuIDStr != "" {
		parsedSKUID, parseErr := uuid.Parse(skuIDStr)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SKU ID"})
			return
		}
		skuID = &parsedSKUID
	}

	// Check if product exists
	var productExists bool
	err = database.Database.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM product_models WHERE id = $1 AND is_active = true)",
		productID,
	).Scan(&productExists)
	if err != nil {
		fmt.Printf("❌ SubscribeToProductNotification: database error checking product: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !productExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Check if already subscribed (handle NULL sku_id properly)
	var existingID uuid.UUID
	var checkErr error
	if skuID != nil {
		checkQuery := `SELECT id FROM product_notifications WHERE user_id = $1 AND product_id = $2 AND sku_id = $3`
		checkErr = database.Database.QueryRow(checkQuery, userID, productID, *skuID).Scan(&existingID)
	} else {
		checkQuery := `SELECT id FROM product_notifications WHERE user_id = $1 AND product_id = $2 AND sku_id IS NULL`
		checkErr = database.Database.QueryRow(checkQuery, userID, productID).Scan(&existingID)
	}

	if checkErr == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Already subscribed to notifications",
			"id":      existingID,
		})
		return
	}
	
	// If error is not "no rows", it's a real database error
	if checkErr != sql.ErrNoRows {
		fmt.Printf("❌ SubscribeToProductNotification: database error checking subscription - user: %s, product: %s, error: %v\n", userID, productID, checkErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking subscription"})
		return
	}

	// Check if user has a push token before allowing subscription
	var pushToken sql.NullString
	pushTokenErr := database.Database.QueryRow(
		"SELECT push_token FROM users WHERE id = $1",
		userID,
	).Scan(&pushToken)

	if pushTokenErr != nil || !pushToken.Valid || pushToken.String == "" {
		fmt.Printf("⚠️ SubscribeToProductNotification: user %s has no push token - subscription will be created but notifications cannot be sent until push token is set\n", userID)
		// Still allow subscription - they might enable notifications later
		// But warn them that they need to enable push notifications
	}

	// Create notification subscription
	notificationID := uuid.New()
	now := time.Now().UTC()
	var insertErr error
	
	if skuID != nil {
		insertQuery := `INSERT INTO product_notifications (id, user_id, product_id, sku_id, created_at) VALUES ($1, $2, $3, $4, $5)`
		_, insertErr = database.Database.Exec(insertQuery, notificationID, userID, productID, *skuID, now)
	} else {
		insertQuery := `INSERT INTO product_notifications (id, user_id, product_id, sku_id, created_at) VALUES ($1, $2, $3, NULL, $4)`
		_, insertErr = database.Database.Exec(insertQuery, notificationID, userID, productID, now)
	}

	if insertErr != nil {
		fmt.Printf("❌ SubscribeToProductNotification: failed to insert - user: %s, product: %s, sku: %v, error: %v\n", userID, productID, skuID, insertErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to subscribe"})
		return
	}

	response := gin.H{
		"message": "Successfully subscribed to notifications",
		"id":      notificationID,
	}

	// Warn user if they don't have a push token
	if pushTokenErr != nil || !pushToken.Valid || pushToken.String == "" {
		response["warning"] = "Please enable push notifications in your device settings to receive alerts when this product becomes available"
		fmt.Printf("⚠️ SubscribeToProductNotification: subscription created but user %s needs to enable push notifications\n", userID)
	}

	fmt.Printf("✅ SubscribeToProductNotification: success - user: %s, product: %s, notification_id: %s\n", userID, productID, notificationID)
	c.JSON(http.StatusCreated, response)
}

// UnsubscribeFromProductNotification allows a user to unsubscribe from notifications
func UnsubscribeFromProductNotification(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse user ID - AuthMiddleware stores it as string
	var userID uuid.UUID
	var parseErr error
	if userIDStr, ok := userIDInterface.(string); ok {
		userID, parseErr = uuid.Parse(userIDStr)
	} else if userIDParsed, ok := userIDInterface.(uuid.UUID); ok {
		userID = userIDParsed
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}
	
	if parseErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var skuID *uuid.UUID
	if skuIDStr := c.Query("sku_id"); skuIDStr != "" {
		parsedSKUID, parseErr := uuid.Parse(skuIDStr)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SKU ID"})
			return
		}
		skuID = &parsedSKUID
	}

	var result sql.Result
	if skuID != nil {
		deleteQuery := `DELETE FROM product_notifications 
			WHERE user_id = $1 AND product_id = $2 AND sku_id = $3`
		result, err = database.Database.Exec(deleteQuery, userID, productID, *skuID)
	} else {
		deleteQuery := `DELETE FROM product_notifications 
			WHERE user_id = $1 AND product_id = $2 AND sku_id IS NULL`
		result, err = database.Database.Exec(deleteQuery, userID, productID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsubscribe"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully unsubscribed"})
}

// CheckProductNotificationStatus checks if user is subscribed to a product
func CheckProductNotificationStatus(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse user ID - AuthMiddleware stores it as string
	var userID uuid.UUID
	var parseErr error
	if userIDStr, ok := userIDInterface.(string); ok {
		userID, parseErr = uuid.Parse(userIDStr)
	} else if userIDParsed, ok := userIDInterface.(uuid.UUID); ok {
		userID = userIDParsed
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}
	
	if parseErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var skuID *uuid.UUID
	if skuIDStr := c.Query("sku_id"); skuIDStr != "" {
		parsedSKUID, parseErr := uuid.Parse(skuIDStr)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SKU ID"})
			return
		}
		skuID = &parsedSKUID
	}

	var notificationID uuid.UUID
	var checkErr error
	if skuID != nil {
		checkQuery := `SELECT id FROM product_notifications 
			WHERE user_id = $1 AND product_id = $2 AND sku_id = $3`
		checkErr = database.Database.QueryRow(checkQuery, userID, productID, *skuID).Scan(&notificationID)
	} else {
		checkQuery := `SELECT id FROM product_notifications 
			WHERE user_id = $1 AND product_id = $2 AND sku_id IS NULL`
		checkErr = database.Database.QueryRow(checkQuery, userID, productID).Scan(&notificationID)
	}

	err = checkErr
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"subscribed": false})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscribed": true,
		"id":         notificationID,
	})
}

// GetProductNotificationSubscribers (Admin) - Get all users subscribed to a product
func GetProductNotificationSubscribers(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	query := `
		SELECT 
			pn.id,
			pn.user_id,
			pn.sku_id,
			pn.created_at,
			pn.notified_at,
			u.phone,
			u.email,
			u.full_name,
			pm.title as product_title,
			s.sku_code,
			s.size,
			pc.color_name
		FROM product_notifications pn
		JOIN users u ON pn.user_id = u.id
		JOIN product_models pm ON pn.product_id = pm.id
		LEFT JOIN skus s ON pn.sku_id = s.id
		LEFT JOIN product_colors pc ON s.product_color_id = pc.id
		WHERE pn.product_id = $1
		ORDER BY pn.created_at DESC
	`
	rows, err := database.Database.Query(query, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var subscribers []map[string]interface{}
	for rows.Next() {
		var id, userID uuid.UUID
		var skuID sql.NullString
		var createdAt time.Time
		var notifiedAt sql.NullTime
		var phone, email, fullName sql.NullString
		var productTitle string
		var skuCode, size, colorName sql.NullString

		scanErr := rows.Scan(&id, &userID, &skuID, &createdAt, &notifiedAt,
			&phone, &email, &fullName, &productTitle, &skuCode, &size, &colorName)
		if scanErr != nil {
			continue
		}

		subscriber := map[string]interface{}{
			"id":            id,
			"user_id":       userID,
			"created_at":    createdAt,
			"product_title": productTitle,
		}

		if phone.Valid {
			subscriber["phone"] = phone.String
		}
		if email.Valid {
			subscriber["email"] = email.String
		}
		if fullName.Valid {
			subscriber["full_name"] = fullName.String
		}
		if skuID.Valid {
			subscriber["sku_id"] = skuID.String
		}
		if skuCode.Valid {
			subscriber["sku_code"] = skuCode.String
		}
		if size.Valid {
			subscriber["size"] = size.String
		}
		if colorName.Valid {
			subscriber["color_name"] = colorName.String
		}
		if notifiedAt.Valid {
			subscriber["notified_at"] = notifiedAt.Time
		}

		subscribers = append(subscribers, subscriber)
	}

	c.JSON(http.StatusOK, gin.H{
		"subscribers": subscribers,
		"count":       len(subscribers),
	})
}

// GetProductsWithNotifications (Admin) - Get all products with notification requests
func GetProductsWithNotifications(c *gin.Context) {
	query := `
		SELECT 
			pm.id,
			pm.title,
			pm.model_code,
			b.name as brand_name,
			COUNT(DISTINCT pn.id) as notification_count,
			COUNT(DISTINCT CASE WHEN pn.notified_at IS NULL THEN pn.id END) as pending_count,
			COALESCE(SUM(i.available), 0) as total_stock,
			(SELECT url FROM product_images WHERE product_model_id = pm.id ORDER BY position ASC LIMIT 1) as image_url
		FROM product_models pm
		JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN product_notifications pn ON pm.id = pn.product_id
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN inventory i ON s.id = i.sku_id
		WHERE EXISTS (
			SELECT 1 FROM product_notifications WHERE product_id = pm.id
		)
		GROUP BY pm.id, pm.title, pm.model_code, b.name
		ORDER BY notification_count DESC, pending_count DESC
	`
	rows, err := database.Database.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var products []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var title, modelCode, brandName, imageURL sql.NullString
		var notificationCount, pendingCount, totalStock int

		scanErr := rows.Scan(&id, &title, &modelCode, &brandName, &notificationCount, &pendingCount, &totalStock, &imageURL)
		if scanErr != nil {
			continue
		}

		product := map[string]interface{}{
			"id":                 id,
			"notification_count": notificationCount,
			"pending_count":      pendingCount,
			"total_stock":        totalStock,
		}

		if title.Valid {
			product["title"] = title.String
		}
		if modelCode.Valid {
			product["model_code"] = modelCode.String
		}
		if brandName.Valid {
			product["brand_name"] = brandName.String
		}
		if imageURL.Valid {
			product["image_url"] = imageURL.String
		}

		products = append(products, product)
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"count":    len(products),
	})
}

// TriggerNotificationsForProduct checks stock and sends notifications when stock becomes available
func TriggerNotificationsForProduct(productID uuid.UUID, skuID *uuid.UUID) error {
	// Check if product has stock now
	var hasStock bool
	var query string
	var args []interface{}

	if skuID != nil {
		// Check specific SKU
		query = `
			SELECT COALESCE(i.available, 0) > 0
			FROM skus s
			LEFT JOIN inventory i ON s.id = i.sku_id
			WHERE s.id = $1
		`
		args = []interface{}{*skuID}
	} else {
		// Check if any SKU of the product has stock
		query = `
			SELECT EXISTS(
				SELECT 1 FROM skus s
				LEFT JOIN inventory i ON s.id = i.sku_id
				WHERE s.product_model_id = $1 AND COALESCE(i.available, 0) > 0
			)
		`
		args = []interface{}{productID}
	}

	err := database.Database.QueryRow(query, args...).Scan(&hasStock)
	if err != nil {
		return fmt.Errorf("failed to check stock: %w", err)
	}

	if !hasStock {
		return nil // No stock, no notifications to send
	}

	// Find all pending notifications for this product/SKU
	// When a specific SKU becomes available, notify:
	// 1. Users subscribed to that specific SKU (sku_id = specific SKU)
	// 2. Users subscribed to the product in general (sku_id IS NULL)
	var notificationQuery string
	var notificationArgs []interface{}

	if skuID != nil {
		// Check both SKU-specific AND product-level subscriptions
		notificationQuery = `
			SELECT id, user_id FROM product_notifications
			WHERE product_id = $1 
			AND (sku_id = $2 OR sku_id IS NULL)
			AND notified_at IS NULL
		`
		notificationArgs = []interface{}{productID, *skuID}
	} else {
		// Product-level only (no specific SKU)
		notificationQuery = `
			SELECT id, user_id FROM product_notifications
			WHERE product_id = $1 AND sku_id IS NULL AND notified_at IS NULL
		`
		notificationArgs = []interface{}{productID}
	}

	rows, err := database.Database.Query(notificationQuery, notificationArgs...)
	if err != nil {
		return fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []models.ProductNotification
	for rows.Next() {
		var n models.ProductNotification
		if scanErr := rows.Scan(&n.ID, &n.UserID); scanErr != nil {
			continue
		}
		notifications = append(notifications, n)
	}

	if len(notifications) == 0 {
		fmt.Printf("ℹ️ No pending subscriptions found for product %s (SKU: %v)\n", productID, skuID)
		return nil
	}

	fmt.Printf("📬 Found %d pending notification(s) for product %s (SKU: %v)\n", len(notifications), productID, skuID)

	// Mark notifications as sent
	now := time.Now().UTC()
	for _, notification := range notifications {
		updateQuery := `UPDATE product_notifications SET notified_at = $1 WHERE id = $2`
		_, err := database.Database.Exec(updateQuery, now, notification.ID)
		if err != nil {
			fmt.Printf("⚠️ Failed to mark notification %s as sent: %v\n", notification.ID, err)
			continue
		}

		// Get product title for notification
		var productTitle string
		titleErr := database.Database.QueryRow(
			"SELECT title FROM product_models WHERE id = $1",
			productID,
		).Scan(&productTitle)
		if titleErr != nil || productTitle == "" {
			productTitle = "Product"
		}

		// Send push notification to user
		notifyErr := services.SendProductAvailabilityNotification(
			notification.UserID.String(),
			productTitle,
			productID.String(),
		)
		if notifyErr != nil {
			fmt.Printf("⚠️ Failed to send push notification to user %s: %v\n", notification.UserID, notifyErr)
			// Continue anyway - mark as notified even if push fails
		} else {
			fmt.Printf("✅ Push notification sent successfully to user %s for product %s\n", notification.UserID, productID)
		}
	}

	return nil
}
