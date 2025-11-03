package handlers

import (
	"database/sql"
	"encoding/json"
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

// CreateOrder handles POST /api/v1/orders/
func CreateOrder(c *gin.Context) {
	// DETAILED LOGGING - Backend
	fmt.Printf("🚀 BACKEND ORDER CREATION START\n")
	fmt.Printf("📡 Request method: %s\n", c.Request.Method)
	fmt.Printf("📡 Request URL: %s\n", c.Request.URL.String())
	fmt.Printf("📡 Client IP: %s\n", c.ClientIP())

	userID := c.GetString("user_id")
	if userID == "" {
		fmt.Printf("❌ No user_id in context\n")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	fmt.Printf("👤 User ID: %s\n", userID)

	var request struct {
		Items []struct {
			ProductID string  `json:"product_id" binding:"required"`
			SKUID     string  `json:"sku_id" binding:"required"`
			Quantity  int     `json:"quantity" binding:"required,min=1"`
			Price     float64 `json:"price" binding:"required"`
			Size      *string `json:"size"`
			Color     *string `json:"color"`
		} `json:"items" binding:"required"`
		DeliveryAddress struct {
			AddressID  string   `json:"address_id" binding:"required"`
			City       string   `json:"city" binding:"required"`
			Quartier   string   `json:"quartier" binding:"required"`
			Street     *string  `json:"street"`
			Building   *string  `json:"building"`
			Floor      *string  `json:"floor"`
			Apartment  *string  `json:"apartment"`
			Latitude   *float64 `json:"latitude"`
			Longitude  *float64 `json:"longitude"`
		} `json:"delivery_address" binding:"required"`
		DeliveryOption   string  `json:"delivery_option" binding:"required,oneof=pickup delivery"`
		DeliveryZone     *struct {
			QuartierID   string  `json:"quartier_id"`
			QuartierName string  `json:"quartier_name"`
			DeliveryFee  float64 `json:"delivery_fee"`
		} `json:"delivery_zone"`
		TotalAmount      float64 `json:"total_amount" binding:"required"`
		PaymentProof     string  `json:"payment_proof" binding:"required"`
		PromotionalCode  *string `json:"promotional_code"`
		DiscountAmount   float64 `json:"discount_amount"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		fmt.Printf("❌ JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("📦 Request parsed successfully\n")
	fmt.Printf("📦 Items count: %d\n", len(request.Items))
	fmt.Printf("💰 Total amount: %.2f\n", request.TotalAmount)
	fmt.Printf("🎫 Promotional code: %v\n", request.PromotionalCode)
	fmt.Printf("💸 Discount amount: %.2f\n", request.DiscountAmount)
	fmt.Printf("🚚 Delivery option: %s\n", request.DeliveryOption)
	fmt.Printf("📸 Payment proof: %s\n", request.PaymentProof)

	// Validate delivery option
	if request.DeliveryOption != "pickup" && request.DeliveryOption != "delivery" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delivery option"})
		return
	}

	// Start transaction
	fmt.Printf("🔄 Starting database transaction\n")
	tx, err := database.Database.Begin()
	if err != nil {
		fmt.Printf("❌ Failed to start transaction: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()
	fmt.Printf("✅ Transaction started successfully\n")

	// Generate order number
	orderNumber := generateOrderNumber()
	fmt.Printf("📋 Generated order number: %s\n", orderNumber)

	// Create order
	orderID := uuid.New()
	fmt.Printf("🆔 Generated order ID: %s\n", orderID)
	
	orderQuery := `
		INSERT INTO orders (
			id, user_id, order_number, status, total_amount, 
			delivery_option, delivery_address, payment_proof, 
			currency, delivery_zone_quartier_id, delivery_zone_quartier_name, 
			delivery_zone_fee, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	deliveryAddressJSON, err := json.Marshal(request.DeliveryAddress)
	if err != nil {
		fmt.Printf("❌ Failed to serialize delivery address: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize delivery address"})
		return
	}
	fmt.Printf("🏠 Delivery address serialized: %s\n", string(deliveryAddressJSON))

	now := time.Now()
	fmt.Printf("⏰ Creating order with timestamp: %v\n", now)
	
	// Prepare delivery zone data
	var quartierID, quartierName *string
	var deliveryFee float64 = 0
	
	if request.DeliveryZone != nil {
		quartierID = &request.DeliveryZone.QuartierID
		quartierName = &request.DeliveryZone.QuartierName
		deliveryFee = request.DeliveryZone.DeliveryFee
		fmt.Printf("🏘️ Delivery zone: %s (%s) - Fee: %.2f\n", *quartierName, *quartierID, deliveryFee)
	} else {
		fmt.Printf("🏘️ No delivery zone selected\n")
	}

	_, err = tx.Exec(orderQuery,
		orderID, userID, orderNumber, "pending", request.TotalAmount,
		request.DeliveryOption, string(deliveryAddressJSON), request.PaymentProof,
		"MRU", quartierID, quartierName, deliveryFee, now, now,
	)

	if err != nil {
		fmt.Printf("❌ Failed to create order: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}
	fmt.Printf("✅ Order created successfully\n")

	// Create order items and update inventory
	var orderItems []map[string]interface{}
	for _, item := range request.Items {
		fmt.Printf("Processing order item: ProductID=%s, SKUID=%s, Quantity=%d\n", 
			item.ProductID, item.SKUID, item.Quantity)
		
		// Validate product and SKU
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			fmt.Printf("Invalid product ID: %s\n", item.ProductID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
			return
		}

		skuID, err := uuid.Parse(item.SKUID)
		if err != nil {
			fmt.Printf("Invalid SKU ID: %s\n", item.SKUID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SKU ID"})
			return
		}

		// Check if this is a Melhaf item (same ID used for product and SKU in Melhaf)
		var isMelhaf bool
		var melhafColorExists bool
		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM melhaf_colors 
				WHERE id = $1 AND is_active = true
			)`, productID).Scan(&melhafColorExists)
		
		if err == nil && melhafColorExists && productID == skuID {
			isMelhaf = true
			fmt.Printf("✅ Detected Melhaf item: ColorID=%s\n", productID)
		}

		// Handle Melhaf items differently
		if isMelhaf {
			// Check Melhaf inventory
			var currentQuantity int
			var reservedQuantity int
			err = tx.QueryRow(`
				SELECT COALESCE(available, 0), COALESCE(reserved, 0) 
				FROM melhaf_inventory 
				WHERE color_id = $1`, productID).Scan(&currentQuantity, &reservedQuantity)
			
			if err != nil {
				if err == sql.ErrNoRows {
					fmt.Printf("❌ No Melhaf inventory found for ColorID=%s\n", productID)
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "No inventory found for this Melhaf color",
						"product_id": item.ProductID,
					})
					return
				}
				fmt.Printf("❌ Error checking Melhaf inventory: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check Melhaf inventory"})
				return
			}

			fmt.Printf("📦 Melhaf inventory: ColorID=%s, Available=%d, Reserved=%d, Requested=%d\n", 
				productID, currentQuantity, reservedQuantity, item.Quantity)

			if currentQuantity < item.Quantity {
				fmt.Printf("❌ Insufficient Melhaf quantity: Available=%d, Requested=%d\n", currentQuantity, item.Quantity)
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Insufficient quantity available for this Melhaf color",
					"product_id": item.ProductID,
					"available": currentQuantity,
					"requested": item.Quantity,
				})
				return
			}

			// Update Melhaf inventory
			fmt.Printf("🔄 Updating Melhaf inventory: ColorID=%s, reducing by %d\n", productID, item.Quantity)
			result, err := tx.Exec(`
				UPDATE melhaf_inventory 
				SET available = available - $1, updated_at = $2
				WHERE color_id = $3 AND available >= $1`, item.Quantity, now, productID)
			
			if err != nil {
				fmt.Printf("❌ Failed to update Melhaf inventory: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Melhaf inventory"})
				return
			}
			
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				fmt.Printf("❌ Failed to get rows affected: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify Melhaf inventory update"})
				return
			}
			
			fmt.Printf("✅ Melhaf inventory updated: %d rows affected\n", rowsAffected)
			
			if rowsAffected == 0 {
				fmt.Printf("❌ No rows affected in Melhaf inventory update\n")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Insufficient Melhaf inventory",
					"product_id": item.ProductID,
				})
				return
			}

			// Create order item for Melhaf (use NULL for product_id since it's not in product_models)
			orderItemID := uuid.New()
			fmt.Printf("🆔 Creating Melhaf order item: ID=%s, OrderID=%s, ColorID=%s\n", 
				orderItemID, orderID, productID)
			
			// Get Melhaf color details
			var melhafName, collectionName sql.NullString
			var melhafPrice float64
			err = tx.QueryRow(`
				SELECT mc.name, mc.price, mcol.name as collection_name
				FROM melhaf_colors mc
				JOIN melhaf_collections mcol ON mc.collection_id = mcol.id
				WHERE mc.id = $1`, productID).Scan(&melhafName, &melhafPrice, &collectionName)
			
			if err != nil {
				fmt.Printf("❌ Failed to get Melhaf color details: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Melhaf color details"})
				return
			}

			// Insert order item with NULL product_id for Melhaf items
			// We'll store the Melhaf color ID in sku_id field
			orderItemQuery := `
				INSERT INTO order_items (
					id, order_id, product_id, sku_id, quantity, 
					unit_price, total_price, size, color, created_at
				) VALUES ($1, $2, NULL, $3, $4, $5, $6, NULL, $7, $8)`

			totalPrice := item.Price * float64(item.Quantity)
			fmt.Printf("💰 Melhaf order item prices: Unit=%.2f, Quantity=%d, Total=%.2f\n", 
				item.Price, item.Quantity, totalPrice)

			_, err = tx.Exec(orderItemQuery,
				orderItemID, orderID, productID, item.Quantity, // Use Melhaf color ID as sku_id (productID), quantity as $4
				item.Price, totalPrice, melhafName.String, now,
			)

			if err != nil {
				fmt.Printf("❌ Failed to create Melhaf order item: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Melhaf order item"})
				return
			}
			
			fmt.Printf("✅ Melhaf order item created successfully\n")

			// Add to order items response
			orderItems = append(orderItems, map[string]interface{}{
				"id":           orderItemID.String(),
				"product_id":   nil, // NULL for Melhaf
				"sku_id":       productID.String(),
				"quantity":     item.Quantity,
				"unit_price":   item.Price,
				"total_price":  totalPrice,
				"product_name": fmt.Sprintf("%s - %s", collectionName.String, melhafName.String),
				"brand_name":   "Melhaf",
				"color":        melhafName.String,
			})
			
			continue // Skip regular product processing
		}

		// Regular product processing
		// Check if product exists and is active
		var productExists bool
		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM product_models 
				WHERE id = $1 AND is_active = true
			)`, productID).Scan(&productExists)
		
		if err != nil || !productExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found or inactive"})
			return
		}

		// Check SKU exists and belongs to product
		var skuExists bool
		err = tx.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM skus s
				JOIN product_models pm ON s.product_model_id = pm.id
				WHERE s.id = $1 AND pm.id = $2
			)`, skuID, productID).Scan(&skuExists)
		
		if err != nil || !skuExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "SKU not found for this product"})
			return
		}

		// Check if inventory record exists for this SKU
		var inventoryExists bool
		err = tx.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM inventory WHERE sku_id = $1)`, skuID).Scan(&inventoryExists)
		
		if err != nil {
			fmt.Printf("❌ Error checking inventory existence: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check inventory"})
			return
		}
		
		fmt.Printf("📦 Inventory record exists for SKUID=%s: %t\n", skuID, inventoryExists)
		
		if !inventoryExists {
			fmt.Printf("❌ No inventory record found for SKUID=%s\n", skuID)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No inventory record found for this product variant",
				"product_id": item.ProductID,
				"sku_id": item.SKUID,
			})
			return
		}

		// Check and update inventory - simplified query
		var currentQuantity int
		var reservedQuantity int
		err = tx.QueryRow(`
			SELECT COALESCE(available, 0), COALESCE(reserved, 0) 
			FROM inventory 
			WHERE sku_id = $1`, skuID).Scan(&currentQuantity, &reservedQuantity)
		
		fmt.Printf("📦 Inventory check: SKUID=%s, ProductID=%s, Available=%d, Reserved=%d, Requested=%d\n", 
			skuID, productID, currentQuantity, reservedQuantity, item.Quantity)
		
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("No inventory found for SKUID=%s, ProductID=%s\n", skuID, productID)
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "No inventory found for this product variant",
					"product_id": item.ProductID,
					"sku_id": item.SKUID,
				})
				return
			}
			fmt.Printf("Inventory check error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check inventory"})
			return
		}

		if currentQuantity < item.Quantity {
			fmt.Printf("❌ Insufficient quantity: Available=%d, Requested=%d\n", currentQuantity, item.Quantity)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Insufficient quantity available",
				"product_id": item.ProductID,
				"available": currentQuantity,
				"requested": item.Quantity,
			})
			return
		}

		fmt.Printf("✅ Quantity check passed: Available=%d >= Requested=%d\n", currentQuantity, item.Quantity)

		// Update inventory with proper validation
		fmt.Printf("🔄 Updating inventory: SKUID=%s, reducing by %d\n", skuID, item.Quantity)
		result, err := tx.Exec(`
			UPDATE inventory 
			SET available = available - $1, updated_at = $2
			WHERE sku_id = $3 AND available >= $1`, item.Quantity, now, skuID)
		
		if err != nil {
			fmt.Printf("❌ Failed to update inventory: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update inventory"})
			return
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			fmt.Printf("❌ Failed to get rows affected: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify inventory update"})
			return
		}
		
		fmt.Printf("✅ Inventory updated: %d rows affected\n", rowsAffected)
		
		if rowsAffected == 0 {
			fmt.Printf("❌ No rows affected in inventory update - insufficient inventory or SKU not found\n")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Insufficient inventory or SKU not found",
				"product_id": item.ProductID,
				"sku_id": item.SKUID,
			})
			return
		}

		// Create order item
		orderItemID := uuid.New()
		fmt.Printf("🆔 Creating order item: ID=%s, OrderID=%s, ProductID=%s, SKUID=%s\n", 
			orderItemID, orderID, productID, skuID)
		
		orderItemQuery := `
			INSERT INTO order_items (
				id, order_id, product_id, sku_id, quantity, 
				unit_price, total_price, size, color, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

		totalPrice := item.Price * float64(item.Quantity)
		fmt.Printf("💰 Order item prices: Unit=%.2f, Quantity=%d, Total=%.2f\n", 
			item.Price, item.Quantity, totalPrice)

		_, err = tx.Exec(orderItemQuery,
			orderItemID, orderID, productID, skuID, item.Quantity,
			item.Price, totalPrice, item.Size, item.Color, now,
		)

		if err != nil {
			fmt.Printf("❌ Failed to create order item: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order item"})
			return
		}
		
		fmt.Printf("✅ Order item created successfully\n")

		// Get product details for response
		var productName, brandName, productImage sql.NullString
		err = tx.QueryRow(`
			SELECT pm.title, b.name, 
			COALESCE(
				(SELECT pi.url FROM product_images pi WHERE pi.product_model_id = pm.id ORDER BY pi.created_at LIMIT 1),
				(SELECT pi.url FROM product_images pi 
				 JOIN product_colors pc ON pi.product_color_id = pc.id 
				 WHERE pc.product_model_id = pm.id AND pi.url IS NOT NULL 
				 ORDER BY pi.created_at LIMIT 1)
			) as image_url
			FROM product_models pm
			LEFT JOIN brands b ON pm.brand_id = b.id
			WHERE pm.id = $1`, productID).Scan(&productName, &brandName, &productImage)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get product details"})
			return
		}

		orderItems = append(orderItems, map[string]interface{}{
			"id": orderItemID.String(),
			"product_id": productID.String(),
			"product_name": productName.String,
			"brand_name": brandName.String,
			"product_image": productImage.String,
			"quantity": item.Quantity,
			"unit_price": item.Price,
			"total_price": item.Price * float64(item.Quantity),
			"size": item.Size,
			"color": item.Color,
		})
	}

	// Commit transaction
	fmt.Printf("💾 Committing transaction\n")
	if err := tx.Commit(); err != nil {
		fmt.Printf("❌ Failed to commit transaction: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit order"})
		return
	}
	fmt.Printf("✅ Transaction committed successfully\n")

	// Send push notification for order creation (async, non-blocking)
	go func() {
		fmt.Printf("🔔 Starting push notification process for order #%s\n", orderNumber)
		
		// Get user's push token and name
		var pushToken, customerName sql.NullString
		err := database.Database.QueryRow(`
			SELECT push_token, COALESCE(full_name, phone, 'Customer') 
			FROM users WHERE id = $1`, userID).Scan(&pushToken, &customerName)
		
		if err != nil {
			fmt.Printf("⚠️ Failed to get user info for push notification (user_id: %s): %v\n", userID, err)
			return
		}

		fmt.Printf("👤 User info retrieved - Name: %s, Has Token: %v\n", customerName.String, pushToken.Valid)

		if pushToken.Valid && pushToken.String != "" {
			tokenPreview := pushToken.String
			if len(tokenPreview) > 20 {
				tokenPreview = tokenPreview[:20] + "..."
			}
			fmt.Printf("📱 Sending push notification to token: %s\n", tokenPreview)
			
			notificationService := services.NewNotificationService()
			err := notificationService.SendOrderCreatedNotification(
				pushToken.String, 
				orderNumber, 
				customerName.String, 
				request.TotalAmount,
			)
			if err != nil {
				fmt.Printf("❌ Failed to send order creation notification: %v\n", err)
			} else {
				fmt.Printf("✅ Order creation notification sent successfully to user %s\n", userID)
			}
		} else {
			fmt.Printf("ℹ️ No push token found for user %s, skipping notification\n", userID)
		}
	}()

	// Return order details
	response := gin.H{
		"success": true,
		"message": "Order created successfully",
		"data": gin.H{
			"id": orderID.String(),
			"order_number": orderNumber,
			"status": "pending",
			"total_amount": request.TotalAmount,
			"delivery_option": request.DeliveryOption,
			"delivery_address": request.DeliveryAddress,
			"items": orderItems,
			"created_at": now.Format(time.RFC3339),
		},
	}
	
	fmt.Printf("🎉 ORDER CREATION SUCCESS\n")
	fmt.Printf("🎉 Order ID: %s\n", orderID.String())
	fmt.Printf("🎉 Order Number: %s\n", orderNumber)
	fmt.Printf("🎉 Items count: %d\n", len(orderItems))
	fmt.Printf("🎉 Total amount: %.2f\n", request.TotalAmount)
	
	c.JSON(http.StatusCreated, response)
}

// GetOrder handles GET /api/v1/orders/:id
func GetOrder(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Get order details
	var order models.Order
	var deliveryAddressJSON string
	// var itemsJSON string

	query := `
		SELECT id, user_id, order_number, status, total_amount, 
			   delivery_option, delivery_address, payment_proof,
			   created_at, updated_at
		FROM orders 
		WHERE id = $1 AND user_id = $2`

	err = database.Database.QueryRow(query, orderID, userID).Scan(
		&order.ID, &order.UserID, &order.OrderNumber, &order.Status,
		&order.TotalAmount, &order.DeliveryOption, &deliveryAddressJSON,
		&order.PaymentProof, &order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		}
		return
	}

	// Parse delivery address
	if err := json.Unmarshal([]byte(deliveryAddressJSON), &order.DeliveryAddress); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse delivery address"})
		return
	}

	// Get order items
	itemsQuery := `
		SELECT oi.id, oi.product_id, oi.sku_id, oi.quantity, oi.unit_price,
			   oi.size, oi.color, pm.title, b.name,
			   COALESCE(
				   (SELECT pi.url FROM product_images pi WHERE pi.product_model_id = pm.id ORDER BY pi.created_at LIMIT 1),
				   (SELECT pi.url FROM product_images pi 
					JOIN product_colors pc ON pi.product_color_id = pc.id 
					WHERE pc.product_model_id = pm.id AND pi.url IS NOT NULL 
					ORDER BY pi.created_at LIMIT 1)
			   ) as image_url
		FROM order_items oi
		JOIN product_models pm ON oi.product_id = pm.id
		LEFT JOIN brands b ON pm.brand_id = b.id
		WHERE oi.order_id = $1`

	rows, err := database.Database.Query(itemsQuery, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order items"})
		return
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var productName, brandName, productImage sql.NullString

		err := rows.Scan(
			&item.ID, &item.ProductID, &item.SKUID, &item.Quantity,
			&item.UnitPrice, &item.Size, &item.Color, &productName,
			&brandName, &productImage,
		)

		if err != nil {
			continue
		}

		item.ProductName = productName.String
		item.BrandName = brandName.String
		item.ProductImage = productImage.String
		items = append(items, item)
	}

	order.Items = items

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": order,
	})
}

// TrackOrder handles public order tracking by order number
func TrackOrder(c *gin.Context) {
	orderNumber := c.Param("orderNumber")
	if orderNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order number is required"})
		return
	}

	fmt.Printf("🔍 TRACK ORDER - Order Number: %s\n", orderNumber)

	// Get order details by order number (public access)
	var order models.Order
	var deliveryAddressJSON string

	query := `
		SELECT id, user_id, order_number, status, total_amount, 
			   delivery_option, delivery_address, payment_proof,
			   created_at, updated_at
		FROM orders 
		WHERE order_number = $1`

	err := database.Database.QueryRow(query, orderNumber).Scan(
		&order.ID, &order.UserID, &order.OrderNumber, &order.Status,
		&order.TotalAmount, &order.DeliveryOption, &deliveryAddressJSON,
		&order.PaymentProof, &order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("❌ Order not found: %s\n", orderNumber)
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			fmt.Printf("❌ Database error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		}
		return
	}

	fmt.Printf("✅ Order found: %s, Status: %s\n", order.OrderNumber, order.Status)

	// Parse delivery address
	if deliveryAddressJSON != "" {
		if err := json.Unmarshal([]byte(deliveryAddressJSON), &order.DeliveryAddress); err != nil {
			fmt.Printf("⚠️ Failed to parse delivery address: %v\n", err)
			// Continue without delivery address
		}
	}

	// Get order items
	itemsQuery := `
		SELECT oi.id, oi.product_id, oi.sku_id, oi.quantity, oi.unit_price,
			   oi.size, oi.color, pm.title, b.name,
			   COALESCE(
				   (SELECT pi.url FROM product_images pi WHERE pi.product_model_id = pm.id ORDER BY pi.created_at LIMIT 1),
				   (SELECT pi.url FROM product_images pi 
					JOIN product_colors pc ON pi.product_color_id = pc.id 
					WHERE pc.product_model_id = pm.id AND pi.url IS NOT NULL 
					ORDER BY pi.created_at LIMIT 1)
			   ) as image_url
		FROM order_items oi
		JOIN product_models pm ON oi.product_id = pm.id
		LEFT JOIN brands b ON pm.brand_id = b.id
		WHERE oi.order_id = $1`

	rows, err := database.Database.Query(itemsQuery, order.ID)
	if err != nil {
		fmt.Printf("❌ Failed to fetch order items: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order items"})
		return
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var productName, brandName, productImage sql.NullString

		err := rows.Scan(
			&item.ID, &item.ProductID, &item.SKUID, &item.Quantity,
			&item.UnitPrice, &item.Size, &item.Color, &productName,
			&brandName, &productImage,
		)

		if err != nil {
			fmt.Printf("❌ Failed to scan order item: %v\n", err)
			continue
		}

		item.ProductName = productName.String
		item.BrandName = brandName.String
		item.ProductImage = productImage.String
		items = append(items, item)
	}

	order.Items = items

	fmt.Printf("🎯 Returning order with %d items\n", len(items))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

// GetUserOrders handles GET /api/v1/orders/
func GetUserOrders(c *gin.Context) {
	fmt.Println("🔵 GetUserOrders called")
	userID := c.GetString("user_id")
	fmt.Printf("🔵 User ID: %s\n", userID)
	if userID == "" {
		fmt.Println("❌ No user_id found - unauthorized")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	status := c.Query("status")
	fmt.Printf("🔵 Query params - Page: %s, Limit: %s, Status: %s\n", pageStr, limitStr, status)

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Build query with delivery address
	query := `
		SELECT o.id, o.order_number, o.status, o.total_amount, o.delivery_option,
			   o.delivery_address, o.created_at, o.updated_at
		FROM orders o
		WHERE o.user_id = $1`
	
	args := []interface{}{userID}
	argIndex := 2

	if status != "" {
		query += fmt.Sprintf(" AND o.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := database.Database.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer rows.Close()

	var orders []map[string]interface{}
	for rows.Next() {
		var orderID, orderNumber, orderStatus string
		var totalAmount float64
		var deliveryOption, deliveryAddressJSON string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&orderID, &orderNumber, &orderStatus,
			&totalAmount, &deliveryOption, &deliveryAddressJSON,
			&createdAt, &updatedAt,
		)

		if err != nil {
			continue
		}

		// Parse delivery address
		var deliveryAddress map[string]interface{}
		if deliveryAddressJSON != "" {
			json.Unmarshal([]byte(deliveryAddressJSON), &deliveryAddress)
		}

		// Get order items - First check if order_items table has any data
		checkQuery := `SELECT COUNT(*) FROM order_items WHERE order_id = $1`
		var itemCount int
		err = database.Database.QueryRow(checkQuery, orderID).Scan(&itemCount)
		if err != nil {
			fmt.Printf("❌ Error checking items count for order %s: %v\n", orderID, err)
		} else {
			fmt.Printf("📊 Order %s has %d items in order_items table\n", orderID, itemCount)
		}

		// Get order items - handle NULL product_id values
		itemsQuery := `
			SELECT oi.id, oi.quantity, oi.unit_price, oi.size, oi.color,
				   COALESCE(pm.title, 'Product') as product_name, 
				   COALESCE(b.name, 'Brand') as brand_name, 
				   COALESCE(pi.url, '') as product_image
			FROM order_items oi
			LEFT JOIN product_models pm ON oi.product_id = pm.id
			LEFT JOIN brands b ON pm.brand_id = b.id
			LEFT JOIN product_images pi ON pm.id = pi.product_model_id AND pi.position = 0
			WHERE oi.order_id = $1
		`
		
		fmt.Printf("🔍 Fetching items for order: %s\n", orderID)
		itemRows, err := database.Database.Query(itemsQuery, orderID)
		useFallback := false
		
		if err != nil {
			fmt.Printf("❌ Error querying items for order %s: %v\n", orderID, err)
			// Try fallback query without JOIN
			fallbackQuery := `SELECT id, quantity, unit_price, size, color FROM order_items WHERE order_id = $1`
			itemRows, err = database.Database.Query(fallbackQuery, orderID)
			if err != nil {
				fmt.Printf("❌ Fallback query also failed for order %s: %v\n", orderID, err)
				continue
			}
			useFallback = true
		}

		var items []map[string]interface{}
		itemCount = 0 // Reset itemCount for this order
		
		for itemRows.Next() {
			if useFallback {
				// Fallback scan for simple query
				var fallbackItemID, fallbackSize, fallbackColor string
				var fallbackQuantity int
				var fallbackUnitPrice float64
				
				err = itemRows.Scan(&fallbackItemID, &fallbackQuantity, &fallbackUnitPrice, &fallbackSize, &fallbackColor)
				if err != nil {
					fmt.Printf("❌ Error scanning fallback item for order %s: %v\n", orderID, err)
					continue
				}

				items = append(items, map[string]interface{}{
					"id": fallbackItemID,
					"product_name": "Product",
					"product_image": "",
					"brand_name": "Brand",
					"price": fallbackUnitPrice,
					"quantity": fallbackQuantity,
					"size": fallbackSize,
					"color": fallbackColor,
				})
			} else {
				// Normal scan with JOIN
				var itemID, size, color, productName, brandName, productImage string
				var quantity int
				var unitPrice float64
				
				err = itemRows.Scan(
					&itemID, &quantity, &unitPrice, &size, &color,
					&productName, &brandName, &productImage,
				)
				if err != nil {
					fmt.Printf("❌ Error scanning item for order %s: %v\n", orderID, err)
					continue
				}

				items = append(items, map[string]interface{}{
					"id": itemID,
					"product_name": productName,
					"product_image": productImage,
					"brand_name": brandName,
					"price": unitPrice,
					"quantity": quantity,
					"size": size,
					"color": color,
				})
			}
			itemCount++
		}
		itemRows.Close()
		fmt.Printf("📦 Found %d items for order %s\n", itemCount, orderID)

		order := map[string]interface{}{
			"id": orderID,
			"order_number": orderNumber,
			"status": orderStatus,
			"total_amount": totalAmount,
			"delivery_option": deliveryOption,
			"delivery_address": deliveryAddress,
			"items": items,
			"created_at": createdAt.Format(time.RFC3339),
			"updated_at": updatedAt.Format(time.RFC3339),
		}

		orders = append(orders, order)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM orders WHERE user_id = $1"
	countArgs := []interface{}{userID}
	
	if status != "" {
		countQuery += " AND status = $2"
		countArgs = append(countArgs, status)
	}

	var total int
	err = database.Database.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total count"})
		return
	}

		fmt.Printf("📦 Returning %d orders to frontend\n", len(orders))
		for i, order := range orders {
			if items, ok := order["items"].([]map[string]interface{}); ok {
				fmt.Printf("📦 Order %d (%s): %d items\n", i+1, order["order_number"], len(items))
			} else {
				fmt.Printf("📦 Order %d (%s): items field is %T\n", i+1, order["order_number"], order["items"])
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": orders,
			"pagination": gin.H{
				"page": page,
				"limit": limit,
				"total": total,
				"pages": (total + limit - 1) / limit,
			},
		})
}

// UploadPaymentProof handles payment proof image uploads to Cloudinary
func UploadPaymentProof(c *gin.Context) {
	fmt.Printf("📸 PAYMENT PROOF UPLOAD START\n")
	
	userID := c.GetString("user_id")
	if userID == "" {
		fmt.Printf("❌ No user_id in context\n")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	fmt.Printf("👤 User ID: %s\n", userID)
	
	file, err := c.FormFile("payment_proof")
	if err != nil {
		fmt.Printf("❌ No payment proof file provided: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No payment proof file provided"})
		return
	}

	fmt.Printf("📁 File received: %s (Size: %d bytes)\n", file.Filename, file.Size)
	
	// Check if Cloudinary is initialized
	if services.Cloudinary == nil {
		fmt.Printf("❌ Cloudinary not initialized\n")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Image upload service not available"})
		return
	}
	
	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		fmt.Printf("❌ Failed to open file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Read file data
	fileData := make([]byte, file.Size)
	_, err = src.Read(fileData)
	if err != nil {
		fmt.Printf("❌ Failed to read file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Upload to Cloudinary
	folder := "payment-proofs"
	uploadResult, err := services.Cloudinary.UploadImageFromBytes(fileData, folder, file.Filename)
	if err != nil {
		fmt.Printf("❌ Cloudinary upload failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload payment proof", "details": err.Error()})
		return
	}

	fmt.Printf("✅ Payment proof uploaded successfully: %s\n", uploadResult.URL)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"url": uploadResult.URL,
		"public_id": uploadResult.PublicID,
		"secure_url": uploadResult.SecureURL,
		"message": "Payment proof uploaded successfully",
	})
}

// generateOrderNumber generates a unique order number
func generateOrderNumber() string {
	now := time.Now()
	return fmt.Sprintf("FMBQ-%d%02d%02d-%d", 
		now.Year(), now.Month(), now.Day(), now.Unix()%10000)
}