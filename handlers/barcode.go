package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"fmbq-server/database"

	"github.com/gin-gonic/gin"
)

// GetProductSKUs handles GET /api/v1/admin/products/:id/skus
func GetProductSKUs(c *gin.Context) {
	productID := c.Param("id")
	fmt.Printf("🔍 SIMPLE EAN FETCH - ProductID: %s\n", productID)
	
	// Get SKUs with EAN codes, prices, and inventory - ENHANCED QUERY
	rows, err := database.Database.Query(`
		SELECT 
			s.id,
			s.sku_code,
			s.ean,
			s.size,
			pc.color_name,
			COALESCE(p.list_price, 0) as list_price,
			COALESCE(p.sale_price, 0) as sale_price,
			COALESCE(i.available, 0) as available_quantity,
			COALESCE(i.reserved, 0) as reserved_quantity
		FROM skus s
		JOIN product_colors pc ON s.product_color_id = pc.id
		LEFT JOIN prices p ON s.id = p.sku_id
		LEFT JOIN inventory i ON s.id = i.sku_id
		WHERE s.product_model_id = $1
		ORDER BY pc.color_name, s.size
	`, productID)

	if err != nil {
		fmt.Printf("❌ Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var skus []map[string]interface{}
	for rows.Next() {
		var skuID, skuCode, ean, size, colorName string
		var listPrice, salePrice, availableQuantity, reservedQuantity float64

		err := rows.Scan(&skuID, &skuCode, &ean, &size, &colorName, 
			&listPrice, &salePrice, &availableQuantity, &reservedQuantity)
		if err != nil {
			fmt.Printf("❌ Scan error: %v\n", err)
			continue
		}

		fmt.Printf("✅ SKU: %s, EAN: %s, Price: %.2f, Stock: %.0f\n", 
			skuCode, ean, listPrice, availableQuantity)

		sku := map[string]interface{}{
			"id": skuID,
			"sku_code": skuCode,
			"ean": ean,
			"size": size,
			"color_name": colorName,
			"list_price": listPrice,
			"sale_price": salePrice,
			"available_quantity": int(availableQuantity),
			"reserved_quantity": int(reservedQuantity),
		}
		skus = append(skus, sku)
	}

	fmt.Printf("🎯 Returning %d SKUs for product %s\n", len(skus), productID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"skus": skus,
	})
}

// ScanBarcode handles POST /api/v1/barcode/scan and /api/v1/admin/barcode/scan
func ScanBarcode(c *gin.Context) {
	var req struct {
		EAN string `json:"ean" binding:"required"`
	}

	fmt.Printf("🔍 Barcode scan request received\n")

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("❌ JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("🔍 Scanning EAN: %s\n", req.EAN)

	// Find product by EAN code
	var skuID, productID, productTitle, brandName, skuCode, size, colorName string
	var availableQuantity int
	var listPrice, salePrice float64

	query := `
		SELECT 
			s.id,
			s.product_model_id,
			pm.title,
			b.name,
			s.sku_code,
			s.size,
			pc.color_name,
			COALESCE(i.available, 0) as available_quantity,
			COALESCE(p.list_price, 0) as list_price,
			COALESCE(p.sale_price, 0) as sale_price
		FROM skus s
		JOIN product_models pm ON s.product_model_id = pm.id
		JOIN brands b ON pm.brand_id = b.id
		JOIN product_colors pc ON s.product_color_id = pc.id
		LEFT JOIN inventory i ON s.id = i.sku_id
		LEFT JOIN prices p ON s.id = p.sku_id
		WHERE s.ean = $1
	`

	err := database.Database.QueryRow(query, req.EAN).Scan(
		&skuID, &productID, &productTitle, &brandName, &skuCode, &size, &colorName,
		&availableQuantity, &listPrice, &salePrice,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Product not found",
			"ean": req.EAN,
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get product images
	var images []string
	imageRows, err := database.Database.Query(`
		SELECT url FROM product_images 
		WHERE product_model_id = $1 
		ORDER BY position ASC
	`, productID)
	
	if err == nil {
		defer imageRows.Close()
		for imageRows.Next() {
			var imageURL string
			if err := imageRows.Scan(&imageURL); err == nil {
				images = append(images, imageURL)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"product": gin.H{
			"id": productID,
			"title": productTitle,
			"brand_name": brandName,
			"images": images,
		},
		"sku": gin.H{
			"id": skuID,
			"sku_code": skuCode,
			"ean": req.EAN,
			"size": size,
			"color_name": colorName,
			"available_quantity": availableQuantity,
			"list_price": listPrice,
			"sale_price": salePrice,
		},
		"scan_time": gin.Mode(),
	})
}

// GenerateBarcodeImage handles GET /api/v1/admin/barcode/generate/:ean
func GenerateBarcodeImage(c *gin.Context) {
	ean := c.Param("ean")
	
	// Validate EAN format (should be 13 digits)
	if len(ean) != 13 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid EAN format"})
		return
	}

	// For now, return the EAN as a simple text response
	// In a real implementation, you would generate an actual barcode image
	c.JSON(http.StatusOK, gin.H{
		"ean": ean,
		"barcode_data": ean,
		"format": "EAN-13",
		"message": "Barcode data ready for printing",
	})
}


