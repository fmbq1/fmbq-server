package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"fmbq-server/database"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetAdminStats returns dashboard statistics
func GetAdminStats(c *gin.Context) {
	stats := make(map[string]interface{})

	// Get total products
	var totalProducts int
	err := database.Database.QueryRow("SELECT COUNT(*) FROM product_models").Scan(&totalProducts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get products count"})
		return
	}
	stats["totalProducts"] = totalProducts

	// Get total users
	var totalUsers int
	err = database.Database.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users count"})
		return
	}
	stats["totalUsers"] = totalUsers

	// Get total orders
	var totalOrders int
	err = database.Database.QueryRow("SELECT COUNT(*) FROM orders").Scan(&totalOrders)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders count"})
		return
	}
	stats["totalOrders"] = totalOrders

	// Get total revenue
	var totalRevenue sql.NullFloat64
	err = database.Database.QueryRow("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status IN ('paid', 'shipped', 'delivered')").Scan(&totalRevenue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get revenue"})
		return
	}
	stats["revenue"] = totalRevenue.Float64

	c.JSON(http.StatusOK, stats)
}

// GetAdminUsers returns all users for admin management
func GetAdminUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")
	role := c.Query("role")

	offset := (page - 1) * limit

	query := `
		SELECT id, email, phone, full_name, role, is_active, created_at, metadata
		FROM users
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		query += ` AND (full_name ILIKE $` + strconv.Itoa(argIndex) + ` OR email ILIKE $` + strconv.Itoa(argIndex) + ` OR phone ILIKE $` + strconv.Itoa(argIndex) + `)`
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
		argIndex += 3
	}

	if role != "" {
		query += ` AND role = $` + strconv.Itoa(argIndex)
		args = append(args, role)
		argIndex++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	rows, err := database.Database.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var user struct {
			ID        uuid.UUID `json:"id"`
			Email     *string   `json:"email"`
			Phone     *string   `json:"phone"`
			FullName  *string   `json:"full_name"`
			Role      string    `json:"role"`
			IsActive  bool      `json:"is_active"`
			CreatedAt string    `json:"created_at"`
			Metadata  string    `json:"metadata"`
		}

		err := rows.Scan(
			&user.ID, &user.Email, &user.Phone, &user.FullName,
			&user.Role, &user.IsActive, &user.CreatedAt, &user.Metadata,
		)
		if err != nil {
			continue
		}

		users = append(users, gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"phone":      user.Phone,
			"full_name":  user.FullName,
			"role":       user.Role,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
			"metadata":   user.Metadata,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"page":  page,
		"limit": limit,
	})
}


// GetAdminOrders returns all orders for admin management
func GetAdminOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	status := c.Query("status")

	offset := (page - 1) * limit

	fmt.Printf("🔍 ADMIN ORDERS - Page: %d, Limit: %d, Status: %s\n", page, limit, status)

	query := `
		SELECT 
			o.id, o.order_number, o.user_id, o.status, o.payment_status, o.delivery_option,
			o.total_amount, o.currency, o.delivery_address, o.payment_proof, o.created_at, o.updated_at,
			o.delivery_zone_quartier_id, o.delivery_zone_quartier_name, o.delivery_zone_fee,
			u.first_name, u.last_name, u.email, u.phone
		FROM orders o
		LEFT JOIN users u ON o.user_id = u.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if status != "" && status != "all" {
		query += ` AND o.status = $` + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	query += ` ORDER BY o.created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	rows, err := database.Database.Query(query, args...)
	if err != nil {
		fmt.Printf("❌ Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer rows.Close()

	var orders []gin.H
	for rows.Next() {
		var orderID, orderNumber, status, paymentStatus, deliveryOption, currency, paymentProof string
		var totalAmount, deliveryZoneFee float64
		var deliveryAddress sql.NullString
		var createdAt, updatedAt time.Time
		var firstName, lastName, email, phone sql.NullString
		var deliveryZoneQuartierID, deliveryZoneQuartierName sql.NullString
		var userID sql.NullString

		err := rows.Scan(
			&orderID, &orderNumber, &userID, &status, &paymentStatus, &deliveryOption,
			&totalAmount, &currency, &deliveryAddress, &paymentProof, &createdAt, &updatedAt,
			&deliveryZoneQuartierID, &deliveryZoneQuartierName, &deliveryZoneFee,
			&firstName, &lastName, &email, &phone,
		)
		if err != nil {
			fmt.Printf("❌ Scan error: %v\n", err)
			continue
		}

		// Parse delivery address
		var deliveryAddr map[string]interface{}
		if deliveryAddress.Valid && deliveryAddress.String != "" {
			json.Unmarshal([]byte(deliveryAddress.String), &deliveryAddr)
		}

		// Build customer name
		customerName := ""
		if firstName.Valid && lastName.Valid {
			customerName = firstName.String + " " + lastName.String
		} else if firstName.Valid {
			customerName = firstName.String
		}

		// Build delivery zone info
		deliveryZone := gin.H{}
		if deliveryZoneQuartierID.Valid && deliveryZoneQuartierName.Valid {
			deliveryZone = gin.H{
				"quartier_id":   deliveryZoneQuartierID.String,
				"quartier_name": deliveryZoneQuartierName.String,
				"delivery_fee":  deliveryZoneFee,
			}
		}

		order := gin.H{
			"id":               orderID,
			"order_number":     orderNumber,
			"user_id":          userID.String,
			"status":           status,
			"payment_status":   paymentStatus,
			"delivery_option":  deliveryOption,
			"total_amount":     totalAmount,
			"currency":         currency,
			"delivery_address": deliveryAddr,
			"delivery_zone":    deliveryZone,
			"payment_proof":    paymentProof,
			"created_at":       createdAt.Format(time.RFC3339),
			"updated_at":       updatedAt.Format(time.RFC3339),
			"customer_name":    customerName,
			"customer_email":   email.String,
			"customer_phone":   phone.String,
		}

		// Get order items
		items, err := getOrderItemsForAdmin(orderID)
		if err != nil {
			fmt.Printf("❌ Error getting order items for order %s: %v\n", orderID, err)
		} else {
			order["items"] = items
		}

		orders = append(orders, order)
	}

	fmt.Printf("✅ Found %d orders\n", len(orders))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"orders":  orders,
		"page":    page,
		"limit":   limit,
		"total":   len(orders),
	})
}

// getOrderItemsForAdmin retrieves items for a specific order
func getOrderItemsForAdmin(orderID string) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			oi.id,
			oi.product_id,
			oi.sku_id,
			oi.quantity,
			oi.unit_price,
			oi.total_price,
			oi.size,
			oi.color,
			pm.title as product_name,
			b.name as brand_name,
			pi.url as image_url
		FROM order_items oi
		LEFT JOIN product_models pm ON oi.product_id = pm.id
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN product_images pi ON pm.id = pi.product_model_id AND pi.position = 1
		WHERE oi.order_id = $1
		ORDER BY oi.created_at
	`

	rows, err := database.Database.Query(query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []map[string]interface{}

	for rows.Next() {
		var itemID, productID, skuID, size, color, productName, brandName string
		var quantity int
		var unitPrice, totalPrice float64
		var imageURL sql.NullString

		err := rows.Scan(
			&itemID, &productID, &skuID, &quantity, &unitPrice, &totalPrice,
			&size, &color, &productName, &brandName, &imageURL,
		)

		if err != nil {
			fmt.Printf("❌ Item scan error: %v\n", err)
			continue
		}

		item := map[string]interface{}{
			"id":           itemID,
			"product_id":   productID,
			"sku_id":       skuID,
			"quantity":     quantity,
			"unit_price":   unitPrice,
			"total_price":  totalPrice,
			"size":         size,
			"color":        color,
			"product_name": productName,
			"brand_name":   brandName,
			"image_url":    imageURL.String,
		}

		items = append(items, item)
	}

	return items, nil
}

// UpdateOrderStatus updates an order's status
func UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("🔄 UPDATING ORDER STATUS - OrderID: %s, New Status: %s\n", orderID, req.Status)

	// Validate status
	validStatuses := []string{"pending", "confirmed", "processing", "shipped", "delivered", "cancelled", "returned"}
	validStatus := false
	for _, status := range validStatuses {
		if req.Status == status {
			validStatus = true
			break
		}
	}
	
	if !validStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	query := `UPDATE orders SET status = $1, updated_at = now() WHERE id = $2`
	result, err := database.Database.Exec(query, req.Status, orderID)
	if err != nil {
		fmt.Printf("❌ Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("❌ Rows affected error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	fmt.Printf("✅ Order status updated successfully - Rows affected: %d\n", rowsAffected)

	// Send push notification for order status change
	go func() {
		// Get order details and user's push token
		var orderNumber, customerName sql.NullString
		var pushToken sql.NullString
		
		err := database.Database.QueryRow(`
			SELECT o.order_number, 
				   COALESCE(u.full_name, 'Customer') as customer_name,
				   u.push_token
			FROM orders o
			LEFT JOIN users u ON o.user_id = u.id
			WHERE o.id = $1`, orderID).Scan(&orderNumber, &customerName, &pushToken)
		
		if err != nil {
			fmt.Printf("⚠️ Failed to get order details for push notification: %v\n", err)
			return
		}

		if pushToken.Valid && pushToken.String != "" && orderNumber.Valid {
			notificationService := services.NewNotificationService()
			err := notificationService.SendOrderStatusNotification(
				pushToken.String,
				orderNumber.String,
				req.Status,
				customerName.String,
			)
			if err != nil {
				fmt.Printf("⚠️ Failed to send order status notification: %v\n", err)
			} else {
				fmt.Printf("✅ Order status notification sent successfully\n")
			}
		} else {
			fmt.Printf("ℹ️ No push token or order number found, skipping notification\n")
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Order status updated successfully",
		"order_id": orderID,
		"new_status": req.Status,
	})
}

// GetOrderDetails returns detailed order information for admin
func GetOrderDetails(c *gin.Context) {
	orderID := c.Param("id")
	
	// Get order with customer info
	var order struct {
		ID                   uuid.UUID `json:"id"`
		OrderNumber          string    `json:"order_number"`
		Status               string    `json:"status"`
		TotalAmount          float64   `json:"total_amount"`
		Currency             string    `json:"currency"`
		CreatedAt            string    `json:"created_at"`
		UpdatedAt            string    `json:"updated_at"`
		CustomerName         *string   `json:"customer_name"`
		CustomerEmail        *string   `json:"customer_email"`
		CustomerPhone        *string   `json:"customer_phone"`
		ShippingAddressID    *string   `json:"shipping_address_id"`
		BillingAddressID     *string   `json:"billing_address_id"`
	}
	
	query := `
		SELECT o.id, o.order_number, o.status, o.total_amount, o.currency, 
		       o.created_at, o.updated_at, o.shipping_address_id, o.billing_address_id,
		       u.full_name, u.email, u.phone
		FROM orders o
		LEFT JOIN users u ON o.user_id = u.id
		WHERE o.id = $1
	`
	
	err := database.Database.QueryRow(query, orderID).Scan(
		&order.ID, &order.OrderNumber, &order.Status, &order.TotalAmount,
		&order.Currency, &order.CreatedAt, &order.UpdatedAt,
		&order.ShippingAddressID, &order.BillingAddressID,
		&order.CustomerName, &order.CustomerEmail, &order.CustomerPhone,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		}
		return
	}

	// Get order items
	itemsQuery := `
		SELECT oi.id, oi.quantity, oi.unit_price, oi.total_price,
		       s.sku_code, s.size, s.size_normalized,
		       pm.title as product_title,
		       pc.color_name, pc.color_code,
		       b.name as brand_name
		FROM order_items oi
		JOIN skus s ON oi.sku_id = s.id
		JOIN product_models pm ON s.product_model_id = pm.id
		JOIN product_colors pc ON s.product_color_id = pc.id
		JOIN brands b ON pm.brand_id = b.id
		WHERE oi.order_id = $1
		ORDER BY oi.id
	`
	
	rows, err := database.Database.Query(itemsQuery, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order items"})
		return
	}
	defer rows.Close()

	var items []gin.H
	for rows.Next() {
		var item struct {
			ID              uuid.UUID `json:"id"`
			Quantity        int       `json:"quantity"`
			UnitPrice       float64   `json:"unit_price"`
			TotalPrice      float64   `json:"total_price"`
			SKUCode         string    `json:"sku_code"`
			Size            *string   `json:"size"`
			SizeNormalized  *string   `json:"size_normalized"`
			ProductTitle    string    `json:"product_title"`
			ColorName       string    `json:"color_name"`
			ColorCode       *string   `json:"color_code"`
			BrandName       string    `json:"brand_name"`
		}

		err := rows.Scan(
			&item.ID, &item.Quantity, &item.UnitPrice, &item.TotalPrice,
			&item.SKUCode, &item.Size, &item.SizeNormalized,
			&item.ProductTitle, &item.ColorName, &item.ColorCode, &item.BrandName,
		)
		if err != nil {
			continue
		}

		items = append(items, gin.H{
			"id":               item.ID,
			"quantity":         item.Quantity,
			"unit_price":       item.UnitPrice,
			"total_price":      item.TotalPrice,
			"sku_code":         item.SKUCode,
			"size":             item.Size,
			"size_normalized":  item.SizeNormalized,
			"product_title":    item.ProductTitle,
			"color_name":       item.ColorName,
			"color_code":       item.ColorCode,
			"brand_name":       item.BrandName,
		})
	}

	orderData := gin.H{
		"id":                   order.ID,
		"order_number":         order.OrderNumber,
		"status":               order.Status,
		"total_amount":         order.TotalAmount,
		"currency":             order.Currency,
		"created_at":           order.CreatedAt,
		"updated_at":           order.UpdatedAt,
		"customer_name":        order.CustomerName,
		"customer_email":       order.CustomerEmail,
		"customer_phone":       order.CustomerPhone,
		"shipping_address_id":  order.ShippingAddressID,
		"billing_address_id":   order.BillingAddressID,
		"items":                items,
	}

	c.JSON(http.StatusOK, orderData)
}
