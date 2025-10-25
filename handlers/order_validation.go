package handlers

import (
	"fmt"
	"net/http"

	"fmbq-server/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ValidateProductVariant handles POST /api/v1/products/:id/validate-variant
func ValidateProductVariant(c *gin.Context) {
	productIDStr := c.Param("id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var request struct {
		SKUID    string  `json:"sku_id" binding:"required"`
		Color    *string `json:"color"`
		Size     *string `json:"size"`
		Quantity int     `json:"quantity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("🔍 Validating product %s with SKU %s\n", productIDStr, request.SKUID)

	// Validate that the product exists (remove is_active check for now)
	var productExists bool
	err = database.Database.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM product_models 
			WHERE id = $1
		)`, productID).Scan(&productExists)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate product"})
		return
	}

	if !productExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Validate SKU exists and belongs to the product
	var skuExists bool
	var availableQuantity int
	var price float64
	var originalPrice float64

	query := `
		SELECT EXISTS(
			SELECT 1 FROM skus s
			JOIN product_models pm ON s.product_model_id = pm.id
			WHERE s.id = $1 AND pm.id = $2 AND pm.is_active = true
		), 
		COALESCE(i.available, 0) as available_quantity,
		COALESCE(p.sale_price, 0) as price,
		COALESCE(p.list_price, 0) as original_price
		FROM skus s
		JOIN product_models pm ON s.product_model_id = pm.id
		LEFT JOIN inventory i ON s.id = i.sku_id
		LEFT JOIN prices p ON s.id = p.sku_id
		WHERE s.id = $1 AND pm.id = $2 AND pm.is_active = true
		ORDER BY p.created_at DESC
		LIMIT 1`

	err = database.Database.QueryRow(query, request.SKUID, productID).Scan(
		&skuExists, &availableQuantity, &price, &originalPrice,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate SKU"})
		return
	}

	if !skuExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "SKU not found for this product"})
		return
	}

	// Check if requested quantity is available (but don't fail, just warn)
	if availableQuantity < request.Quantity {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"valid": false,
			"warning": true,
			"error": fmt.Sprintf("Only %d available, requested %d", availableQuantity, request.Quantity),
			"available_quantity": availableQuantity,
			"requested_quantity": request.Quantity,
			"price": price,
			"original_price": originalPrice,
			"message": "Quantity may be limited, but order can still be processed",
		})
		return
	}

	// Validate color if provided
	if request.Color != nil && *request.Color != "" {
		var colorExists bool
		err = database.Database.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM product_colors pc
				JOIN skus s ON pc.product_model_id = s.product_model_id
				WHERE s.id = $1 AND pc.color_name = $2
			)`, request.SKUID, *request.Color).Scan(&colorExists)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate color"})
			return
		}

		if !colorExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Color not available for this product"})
			return
		}
	}

	// Validate size if provided
	if request.Size != nil && *request.Size != "" {
		var sizeExists bool
		err = database.Database.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM size_charts sc
				JOIN skus s ON sc.id = s.size_chart_id
				WHERE s.id = $1 AND sc.size_name = $2
			)`, request.SKUID, *request.Size).Scan(&sizeExists)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate size"})
			return
		}

		if !sizeExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Size not available for this product"})
			return
		}
	}

	// Return validation success with current pricing
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"valid": true,
		"available_quantity": availableQuantity,
		"price": price,
		"original_price": originalPrice,
		"message": "Product variant is valid and available",
	})
}

// ValidateCartItems handles POST /api/v1/cart/validate
func ValidateCartItems(c *gin.Context) {
	var request struct {
		Items []struct {
			ProductID string  `json:"product_id" binding:"required"`
			SKUID     string  `json:"sku_id" binding:"required"`
			Color     *string `json:"color"`
			Size      *string `json:"size"`
			Quantity  int     `json:"quantity" binding:"required,min=1"`
		} `json:"items" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var validationResults []map[string]interface{}

	for _, item := range request.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			validationResults = append(validationResults, map[string]interface{}{
				"product_id": item.ProductID,
				"valid": false,
				"error": "Invalid product ID",
			})
			continue
		}

		// Validate product exists
		var productExists bool
		err = database.Database.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM product_models 
				WHERE id = $1
			)`, productID).Scan(&productExists)
		
		if err != nil || !productExists {
			validationResults = append(validationResults, map[string]interface{}{
				"product_id": item.ProductID,
				"valid": false,
				"error": "Product not found or inactive",
			})
			continue
		}

		// Validate SKU and availability
		var skuExists bool
		var availableQuantity int
		var price float64
		var originalPrice float64

		query := `
			SELECT EXISTS(
				SELECT 1 FROM skus s
				JOIN product_models pm ON s.product_model_id = pm.id
				WHERE s.id = $1 AND pm.id = $2 AND pm.is_active = true
			), 
			COALESCE(i.available, 0) as available_quantity,
			COALESCE(p.sale_price, 0) as price,
			COALESCE(p.list_price, 0) as original_price
			FROM skus s
			JOIN product_models pm ON s.product_model_id = pm.id
			LEFT JOIN inventory i ON s.id = i.sku_id
			LEFT JOIN prices p ON s.id = p.sku_id
			WHERE s.id = $1 AND pm.id = $2 AND pm.is_active = true
			ORDER BY p.created_at DESC
			LIMIT 1`

		err = database.Database.QueryRow(query, item.SKUID, productID).Scan(
			&skuExists, &availableQuantity, &price, &originalPrice,
		)

		if err != nil || !skuExists {
			validationResults = append(validationResults, map[string]interface{}{
				"product_id": item.ProductID,
				"valid": false,
				"error": "SKU not found for this product",
			})
			continue
		}

		// Check quantity availability (warn but don't fail)
		if availableQuantity < item.Quantity {
			validationResults = append(validationResults, map[string]interface{}{
				"product_id": item.ProductID,
				"valid": false,
				"warning": true,
				"error": fmt.Sprintf("Only %d available, requested %d", availableQuantity, item.Quantity),
				"available": availableQuantity,
				"requested": item.Quantity,
				"price": price,
				"original_price": originalPrice,
			})
			// Don't set hasErrors = true for quantity issues
			continue
		}

		// Validate color if provided
		if item.Color != nil && *item.Color != "" {
			var colorExists bool
			err = database.Database.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM product_colors pc
					JOIN skus s ON pc.product_model_id = s.product_model_id
					WHERE s.id = $1 AND pc.color_name = $2
				)`, item.SKUID, *item.Color).Scan(&colorExists)
			
			if err != nil || !colorExists {
				validationResults = append(validationResults, map[string]interface{}{
					"product_id": item.ProductID,
					"valid": false,
					"error": "Color not available for this product",
				})
				continue
			}
		}

		// Validate size if provided
		if item.Size != nil && *item.Size != "" {
			var sizeExists bool
			err = database.Database.QueryRow(`
				SELECT EXISTS(
					SELECT 1 FROM size_charts sc
					JOIN skus s ON sc.id = s.size_chart_id
					WHERE s.id = $1 AND sc.size_name = $2
				)`, item.SKUID, *item.Size).Scan(&sizeExists)
			
			if err != nil || !sizeExists {
				validationResults = append(validationResults, map[string]interface{}{
					"product_id": item.ProductID,
					"valid": false,
					"error": "Size not available for this product",
				})
				continue
			}
		}

		// Item is valid
		validationResults = append(validationResults, map[string]interface{}{
			"product_id": item.ProductID,
			"valid": true,
			"available_quantity": availableQuantity,
			"price": price,
			"original_price": originalPrice,
		})
	}

	// Check if there are any critical errors (not just warnings)
	hasCriticalErrors := false
	for _, result := range validationResults {
		if valid, ok := result["valid"].(bool); ok && !valid {
			if warning, ok := result["warning"].(bool); !ok || !warning {
				hasCriticalErrors = true
				break
			}
		}
	}

	statusCode := http.StatusOK
	if hasCriticalErrors {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, gin.H{
		"success": !hasCriticalErrors,
		"results": validationResults,
		"has_errors": hasCriticalErrors,
		"has_warnings": len(validationResults) > 0,
	})
}
