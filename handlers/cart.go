package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetCart(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Get or create cart for user
	var cartID uuid.UUID
	query := `SELECT id FROM carts WHERE user_id = $1`
	err := DB.QueryRow(query, userID).Scan(&cartID)
	
	if err == sql.ErrNoRows {
		// Create new cart
		cartID = uuid.New()
		insertQuery := `INSERT INTO carts (id, user_id) VALUES ($1, $2)`
		_, err = DB.Exec(insertQuery, cartID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cart"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart"})
		return
	}

	// Get cart items with product details
	itemsQuery := `
		SELECT ci.id, ci.quantity, ci.added_at,
		       s.id as sku_id, s.sku_code, s.size, s.size_normalized,
		       pm.id as product_model_id, pm.title as product_title,
		       pc.color_name, pc.color_code,
		       b.name as brand_name,
		       p.list_price, p.sale_price, p.currency,
		       i.available
		FROM cart_items ci
		JOIN skus s ON ci.sku_id = s.id
		JOIN product_models pm ON s.product_model_id = pm.id
		JOIN product_colors pc ON s.product_color_id = pc.id
		JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN prices p ON s.id = p.sku_id AND p.currency = 'MRO'
		LEFT JOIN inventory i ON s.id = i.sku_id
		WHERE ci.cart_id = $1
		ORDER BY ci.added_at DESC
	`
	
	rows, err := DB.Query(itemsQuery, cartID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart items"})
		return
	}
	defer rows.Close()

	var items []gin.H
	totalAmount := 0.0
	
	for rows.Next() {
		var itemID, skuID, productModelID uuid.UUID
		var quantity int
		var addedAt time.Time
		var skuCode, size, sizeNormalized sql.NullString
		var productTitle, colorName, colorCode, brandName sql.NullString
		var listPrice, salePrice sql.NullFloat64
		var currency sql.NullString
		var available sql.NullInt64
		
		err := rows.Scan(
			&itemID, &quantity, &addedAt, &skuID, &skuCode, &size, &sizeNormalized,
			&productModelID, &productTitle, &colorName, &colorCode, &brandName,
			&listPrice, &salePrice, &currency, &available,
		)
		if err != nil {
			continue
		}

		price := 0.0
		if listPrice.Valid {
			price = listPrice.Float64
		}
		if salePrice.Valid && salePrice.Float64 > 0 {
			price = salePrice.Float64
		}

		itemTotal := price * float64(quantity)
		totalAmount += itemTotal

		item := gin.H{
			"id":                itemID,
			"quantity":          quantity,
			"added_at":          addedAt,
			"sku_id":            skuID,
			"sku_code":          skuCode.String,
			"size":              size.String,
			"size_normalized":   sizeNormalized.String,
			"product_model_id":  productModelID,
			"product_title":     productTitle.String,
			"color_name":        colorName.String,
			"color_code":        colorCode.String,
			"brand_name":        brandName.String,
			"list_price":        listPrice.Float64,
			"sale_price":        salePrice.Float64,
			"currency":          currency.String,
			"available":         available.Int64,
			"unit_price":        price,
			"total_price":       itemTotal,
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"cart_id":      cartID,
		"items":        items,
		"total_amount": totalAmount,
		"currency":     "MRO",
	})
}

func AddToCart(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var req struct {
		SKUID    string `json:"sku_id" binding:"required"`
		Quantity int    `json:"quantity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate SKU exists and is available
	var available int
	skuQuery := `SELECT COALESCE(i.available, 0) FROM skus s 
	             LEFT JOIN inventory i ON s.id = i.sku_id 
	             WHERE s.id = $1`
	err := DB.QueryRow(skuQuery, req.SKUID).Scan(&available)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SKU not found"})
		return
	}

	if available < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient inventory"})
		return
	}

	// Get or create cart
	var cartID uuid.UUID
	cartQuery := `SELECT id FROM carts WHERE user_id = $1`
	err = DB.QueryRow(cartQuery, userID).Scan(&cartID)
	
	if err == sql.ErrNoRows {
		cartID = uuid.New()
		insertCartQuery := `INSERT INTO carts (id, user_id) VALUES ($1, $2)`
		_, err = DB.Exec(insertCartQuery, cartID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cart"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart"})
		return
	}

	// Check if item already exists in cart
	var existingQuantity int
	existingQuery := `SELECT quantity FROM cart_items WHERE cart_id = $1 AND sku_id = $2`
	err = DB.QueryRow(existingQuery, cartID, req.SKUID).Scan(&existingQuantity)
	
	if err == nil {
		// Update existing item
		newQuantity := existingQuantity + req.Quantity
		if newQuantity > available {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient inventory for requested quantity"})
			return
		}
		
		updateQuery := `UPDATE cart_items SET quantity = $1 WHERE cart_id = $2 AND sku_id = $3`
		_, err = DB.Exec(updateQuery, newQuantity, cartID, req.SKUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cart item"})
			return
		}
	} else {
		// Add new item
		itemID := uuid.New()
		insertQuery := `INSERT INTO cart_items (id, cart_id, sku_id, quantity) VALUES ($1, $2, $3, $4)`
		_, err = DB.Exec(insertQuery, itemID, cartID, req.SKUID, req.Quantity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item to cart"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item added to cart successfully"})
}

func UpdateCartItem(c *gin.Context) {
	// Implementation for updating cart items
	c.JSON(http.StatusOK, gin.H{"message": "Update cart item endpoint"})
}

func RemoveFromCart(c *gin.Context) {
	// Implementation for removing items from cart
	c.JSON(http.StatusOK, gin.H{"message": "Remove from cart endpoint"})
}

func ClearCart(c *gin.Context) {
	// Implementation for clearing cart
	c.JSON(http.StatusOK, gin.H{"message": "Clear cart endpoint"})
}
