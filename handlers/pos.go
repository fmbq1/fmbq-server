package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fmbq-server/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetPOSCatalog returns products and SKUs filtered by optional category/brand/query/code
func GetPOSCatalog(c *gin.Context) {
	categoryID := c.Query("category_id")
	brandID := c.Query("brand_id")
	productModelID := c.Query("product_model_id")
	colorID := c.Query("color_id")
	size := c.Query("size")
	query := strings.TrimSpace(c.Query("query"))
	code := strings.TrimSpace(c.Query("code"))

	args := []interface{}{}
	conditions := []string{"1=1"}

	if categoryID != "" {
		conditions = append(conditions, "pmc.category_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, categoryID)
	}
	if brandID != "" {
		conditions = append(conditions, "pm.brand_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, brandID)
	}
	if productModelID != "" {
		conditions = append(conditions, "s.product_model_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, productModelID)
	}
	if colorID != "" {
		conditions = append(conditions, "s.product_color_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, colorID)
	}
	if size != "" {
		conditions = append(conditions, "s.size = $"+strconv.Itoa(len(args)+1))
		args = append(args, size)
	}
	if query != "" {
		conditions = append(conditions, "(pm.title ILIKE $"+strconv.Itoa(len(args)+1)+" OR b.name ILIKE $"+strconv.Itoa(len(args)+2)+")")
		args = append(args, "%"+query+"%", "%"+query+"%")
	}
	if code != "" {
		conditions = append(conditions, "(s.sku_code ILIKE $"+strconv.Itoa(len(args)+1)+")")
		args = append(args, "%"+code+"%")
	}

    querySQL := `
        SELECT s.id, s.sku_code, s.size, s.size_normalized, s.product_model_id,
               pm.title,
               pc.id as color_id, pc.color_name, pc.color_code,
               b.id as brand_id, b.name as brand_name,
               COALESCE(p.sale_price, p.list_price, 0) as price,
               COALESCE(i.available, 0) as available,
               img.image_url,
               pmc.category_id
        FROM skus s
        JOIN product_models pm ON s.product_model_id = pm.id
        JOIN product_colors pc ON s.product_color_id = pc.id
        JOIN brands b ON pm.brand_id = b.id
        LEFT JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
        LEFT JOIN prices p ON p.sku_id = s.id
        LEFT JOIN inventory i ON i.sku_id = s.id
        LEFT JOIN LATERAL (
           SELECT url as image_url FROM product_images pi
           WHERE pi.product_model_id = pm.id
           ORDER BY position NULLS LAST, created_at DESC
           LIMIT 1
        ) img ON true
        WHERE ` + strings.Join(conditions, " AND ") + `
        ORDER BY pm.title, b.name, pc.color_name, s.size_normalized
    `

	rows, err := DB.Query(querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load POS catalog"})
		return
	}
	defer rows.Close()

    type item struct {
        SKUId         uuid.UUID `json:"sku_id"`
        SKUCode       sql.NullString
        Size          sql.NullString
        SizeNorm      sql.NullString
        ProductModelId uuid.UUID
        Title         sql.NullString
        ColorId       uuid.UUID
        ColorName     sql.NullString
        ColorCode     sql.NullString
        BrandId       uuid.UUID
        BrandName     sql.NullString
        Price         float64
        Available     int
        ImageURL      sql.NullString
        CategoryID    sql.NullString
    }

	var items []gin.H
    for rows.Next() {
        var it item
        if err := rows.Scan(&it.SKUId, &it.SKUCode, &it.Size, &it.SizeNorm, &it.ProductModelId, &it.Title, &it.ColorId, &it.ColorName, &it.ColorCode, &it.BrandId, &it.BrandName, &it.Price, &it.Available, &it.ImageURL, &it.CategoryID); err != nil {
			continue
		}
		items = append(items, gin.H{
			"sku_id":        it.SKUId,
			"sku_code":      it.SKUCode.String,
			"size":          it.Size.String,
			"size_normalized": it.SizeNorm.String,
            "product_model_id": it.ProductModelId,
			"title":         it.Title.String,
			"color_id":      it.ColorId,
			"color_name":    it.ColorName.String,
			"color_code":    it.ColorCode.String,
			"brand_id":      it.BrandId,
			"brand_name":    it.BrandName.String,
			"price":         it.Price,
            "available":     it.Available,
            "image_url":     it.ImageURL.String,
            "category_id":   it.CategoryID.String,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GetActivePaymentMethods returns active payment methods for POS
func GetActivePaymentMethods(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, name, label, description, logo FROM payment_methods WHERE is_active = true ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load payment methods"})
		return
	}
	defer rows.Close()

	var methods []gin.H
	for rows.Next() {
		var id uuid.UUID
		var name, label, description, logo sql.NullString
		if err := rows.Scan(&id, &name, &label, &description, &logo); err != nil {
			continue
		}
		methods = append(methods, gin.H{
			"id":          id,
			"name":        name.String,
			"label":       label.String,
			"description": description.String,
			"logo":        logo.String,
		})
	}

	c.JSON(http.StatusOK, gin.H{"payment_methods": methods})
}

// GetPOSCustomers returns a minimal, normalized list of customers for POS linking
// Query params: q (search by name/email/phone), limit (default 50)
func GetPOSCustomers(c *gin.Context) {
    q := strings.TrimSpace(c.Query("q"))
    limitStr := c.DefaultQuery("limit", "50")
    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit <= 0 || limit > 200 {
        limit = 50
    }

    // Build search conditions
    var args []interface{}
    conditions := []string{"1=1"}
    if q != "" {
        conditions = append(conditions, "(COALESCE(c.contact_name,'') ILIKE $1 OR COALESCE(c.company_name,'') ILIKE $1 OR COALESCE(c.email,'') ILIKE $1 OR COALESCE(c.phone,'') ILIKE $1)")
        args = append(args, "%"+q+"%")
    }

    query := `
        SELECT c.id,
               COALESCE(c.contact_name, c.company_name, '') AS name,
               COALESCE(c.email, '') AS email,
               COALESCE(c.phone, '') AS phone
        FROM customers c
        WHERE ` + strings.Join(conditions, " AND ") + `
        ORDER BY c.updated_at DESC NULLS LAST, c.created_at DESC
        LIMIT ` + strconv.Itoa(limit)

    rows, err := DB.Query(query, args...)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load customers"})
        return
    }
    defer rows.Close()

    var customers []gin.H
    for rows.Next() {
        var id uuid.UUID
        var name, email, phone string
        if err := rows.Scan(&id, &name, &email, &phone); err != nil {
            continue
        }
        customers = append(customers, gin.H{
            "id":    id,
            "name":  name,
            "email": email,
            "phone": phone,
        })
    }

    c.JSON(http.StatusOK, gin.H{"customers": customers})
}

// GetProductVariants returns available colors and sizes for a product model
func GetProductVariants(c *gin.Context) {
	productModelID := c.Param("product_model_id")
	
	// Get available colors for this product model
	colorsQuery := `
		SELECT DISTINCT pc.id, pc.color_name, pc.color_code
		FROM product_colors pc
		JOIN skus s ON s.product_color_id = pc.id
		WHERE s.product_model_id = $1
		ORDER BY pc.color_name
	`
	
	rows, err := DB.Query(colorsQuery, productModelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load colors"})
		return
	}
	defer rows.Close()
	
	var colors []gin.H
	for rows.Next() {
		var id uuid.UUID
		var colorName, colorCode sql.NullString
		if err := rows.Scan(&id, &colorName, &colorCode); err != nil {
			continue
		}
		colors = append(colors, gin.H{
			"id":         id,
			"color_name": colorName.String,
			"color_code": colorCode.String,
		})
	}
	
	// Get available sizes for this product model
	sizesQuery := `
		SELECT DISTINCT s.size, s.size_normalized
		FROM skus s
		WHERE s.product_model_id = $1 AND s.size IS NOT NULL
		ORDER BY s.size_normalized
	`
	
	rows, err = DB.Query(sizesQuery, productModelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sizes"})
		return
	}
	defer rows.Close()
	
	var sizes []gin.H
	for rows.Next() {
		var size, sizeNormalized sql.NullString
		if err := rows.Scan(&size, &sizeNormalized); err != nil {
			continue
		}
		sizes = append(sizes, gin.H{
			"size":            size.String,
			"size_normalized": sizeNormalized.String,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"colors": colors,
		"sizes":  sizes,
	})
}

// CreatePOSOrder creates an order directly from POS with items and payment details
func CreatePOSOrder(c *gin.Context) {
	var req struct {
		CustomerID       *string `json:"customer_id"`
		Items            []struct {
			SKUID    string  `json:"sku_id"`
			Quantity int     `json:"quantity"`
			UnitPrice float64 `json:"unit_price"`
		} `json:"items"`
		Currency         string  `json:"currency"`
		PaymentMethodID  string  `json:"payment_method_id"`
		TenderedAmount   float64 `json:"tendered_amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Currency == "" { req.Currency = "MRO" }
	if len(req.Items) == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "No items provided"}); return }

	orderID := uuid.New()
	orderNumber := "POS-" + time.Now().Format("20060102-150405")
    var userID *uuid.UUID
    if req.CustomerID != nil && *req.CustomerID != "" {
        // Look up the customer row
        var linkedUserID, email, phone, contactName sql.NullString
        err := DB.QueryRow(`SELECT user_id, email, phone, contact_name FROM customers WHERE id = $1`, *req.CustomerID).Scan(&linkedUserID, &email, &phone, &contactName)
        if err == nil {
            if linkedUserID.Valid && linkedUserID.String != "" {
                if uid, parseErr := uuid.Parse(linkedUserID.String); parseErr == nil {
                    userID = &uid
                }
            } else {
                // Try to find an existing user by email/phone, then link
                var foundUserID sql.NullString
                if email.Valid && email.String != "" {
                    _ = DB.QueryRow(`SELECT id FROM users WHERE email = $1`, email.String).Scan(&foundUserID)
                }
                if (!foundUserID.Valid || foundUserID.String == "") && phone.Valid && phone.String != "" {
                    _ = DB.QueryRow(`SELECT id FROM users WHERE phone = $1`, phone.String).Scan(&foundUserID)
                }
                if foundUserID.Valid && foundUserID.String != "" {
                    if uid, parseErr := uuid.Parse(foundUserID.String); parseErr == nil {
                        userID = &uid
                        // Best-effort link the customer record
                        _, _ = DB.Exec(`UPDATE customers SET user_id = $1 WHERE id = $2`, uid, *req.CustomerID)
                    }
                } else {
                    // Auto-create a user for this customer so orders link properly
                    newUserID := uuid.New()
                    avatarURL := utils.GenerateRandomAvatar()
                    fullName := contactName.String
                    var emailPtr, phonePtr, fullNamePtr *string
                    if email.Valid && email.String != "" { tmp := email.String; emailPtr = &tmp }
                    if phone.Valid && phone.String != "" { tmp := phone.String; phonePtr = &tmp }
                    if fullName != "" { tmp := fullName; fullNamePtr = &tmp }

                    _, createErr := DB.Exec(`INSERT INTO users (id, email, phone, full_name, avatar, role, is_active, created_at, metadata) 
                                              VALUES ($1,$2,$3,$4,$5,'user',true, now(), '{}')`,
                                              newUserID, emailPtr, phonePtr, fullNamePtr, avatarURL)
                    if createErr == nil {
                        _, _ = DB.Exec(`UPDATE customers SET user_id = $1 WHERE id = $2`, newUserID, *req.CustomerID)
                        userID = &newUserID
                    }
                }
            }
        }
    }

	pmID, err := uuid.Parse(req.PaymentMethodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment method id"})
		return
	}

	// Calculate totals
	total := 0.0
	for _, it := range req.Items { total += float64(it.Quantity) * it.UnitPrice }
	change := req.TenderedAmount - total
	if change < 0 { change = 0 }

	tx, err := DB.Begin()
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction", "details": err.Error()}); return }
	defer tx.Rollback()

	// Insert order
	insertOrder := `INSERT INTO orders (id, user_id, order_number, status, total_amount, currency, payment_method_id, tendered_amount, change_due, created_at, updated_at, source) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now(), 'pos')`
    _, err = tx.Exec(insertOrder, orderID, userID, orderNumber, "paid", total, req.Currency, pmID, req.TenderedAmount, change)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order", "details": err.Error()}); return }

	// Insert items and update inventory
	for _, it := range req.Items {
		skuID, err := uuid.Parse(it.SKUID)
		if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sku id"}); return }
		
		// Check available inventory before updating
		var available int
		checkQuery := `SELECT available FROM inventory WHERE sku_id = $1`
		err = tx.QueryRow(checkQuery, skuID).Scan(&available)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check inventory"})
			return
		}
		
		if available < it.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient inventory for SKU"})
			return
		}
		
		itemID := uuid.New()
		totalPrice := float64(it.Quantity) * it.UnitPrice
        _, err = tx.Exec(`INSERT INTO order_items (id, order_id, sku_id, quantity, unit_price, total_price) VALUES ($1,$2,$3,$4,$5,$6)`, itemID, orderID, skuID, it.Quantity, it.UnitPrice, totalPrice)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order item", "details": err.Error(), "sku_id": skuID}); return }
		
		// Update inventory with atomic operation
        res, err := tx.Exec(`UPDATE inventory SET available = available - $1, reserved = reserved + $1 WHERE sku_id = $2 AND available >= $1`, it.Quantity, skuID)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update inventory", "details": err.Error(), "sku_id": skuID}); return }
        rows, _ := res.RowsAffected()
        if rows == 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient inventory or SKU not found", "sku_id": skuID})
            return
        }
	}

    if err := tx.Commit(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit order", "details": err.Error()}); return }

	c.JSON(http.StatusCreated, gin.H{
		"order_id":     orderID,
		"order_number": orderNumber,
		"status":       "paid",
		"total_amount": total,
		"currency":     req.Currency,
		"change_due":   change,
	})
}

// no-op helper removed; using strconv.Itoa directly above


