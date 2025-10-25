package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fmbq-server/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetAdminProducts retrieves all products for admin management (including inactive)
func GetAdminProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	categoryID := c.Query("category_id")
	brandID := c.Query("brand_id")
	search := c.Query("search")
	status := c.Query("status") // "active", "inactive", or "all"

	offset := (page - 1) * limit

	query := `
		SELECT pm.id, pm.brand_id, pm.title, pm.description, pm.short_description, 
		       pm.model_code, pm.is_active, pm.attributes, pm.created_at, pm.updated_at,
		       b.name as brand_name,
		       COALESCE(MIN(p.list_price), 0) as min_price,
		       COALESCE(MAX(p.list_price), 0) as max_price,
		       COALESCE(SUM(i.available), 0) as total_stock,
		       COUNT(DISTINCT s.id) as variants_count
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN prices p ON s.id = p.sku_id
		LEFT JOIN inventory i ON s.id = i.sku_id
		WHERE 1=1
		GROUP BY pm.id, pm.brand_id, pm.title, pm.description, pm.short_description, 
		         pm.model_code, pm.is_active, pm.attributes, pm.created_at, pm.updated_at,
		         b.name
	`
	args := []interface{}{}
	argIndex := 1

	// Filter by status
	if status == "active" {
		query += ` AND pm.is_active = true`
	} else if status == "inactive" {
		query += ` AND pm.is_active = false`
	}
	// If status is "all" or not provided, show all products

	if categoryID != "" {
		query += ` AND pm.id IN (
			SELECT pmc.product_model_id 
			FROM product_model_categories pmc 
			WHERE pmc.category_id = $` + strconv.Itoa(argIndex) + `
		)`
		args = append(args, categoryID)
		argIndex++
	}

	if brandID != "" {
		query += ` AND pm.brand_id = $` + strconv.Itoa(argIndex)
		args = append(args, brandID)
		argIndex++
	}

	if search != "" {
		query += ` AND (pm.title ILIKE $` + strconv.Itoa(argIndex) + ` OR pm.description ILIKE $` + strconv.Itoa(argIndex) + `)`
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	query += ` ORDER BY pm.created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer rows.Close()

	var products []gin.H
	for rows.Next() {
		var product gin.H
		var id, brandID, title, description, shortDescription, modelCode string
		var isActive bool
		var attributes, createdAt, updatedAt string
		var brandName sql.NullString
		var minPrice, maxPrice float64
		var totalStock int
		var variantsCount int

		err := rows.Scan(&id, &brandID, &title, &description, &shortDescription, 
			&modelCode, &isActive, &attributes, &createdAt, &updatedAt, &brandName,
			&minPrice, &maxPrice, &totalStock, &variantsCount)
		if err != nil {
			continue
		}

		// Get colors and images for this product
		var colors []gin.H
		colorQuery := `SELECT id, color_name, color_code FROM product_colors WHERE product_model_id = $1 ORDER BY created_at`
		colorRows, err := DB.Query(colorQuery, id)
		if err == nil {
			for colorRows.Next() {
				var colorID, colorName, colorCode string
				if err := colorRows.Scan(&colorID, &colorName, &colorCode); err == nil {
					// Get images for this color
					var images []gin.H
					imageQuery := `SELECT url, alt, position FROM product_images WHERE product_color_id = $1 ORDER BY position`
					imageRows, err := DB.Query(imageQuery, colorID)
					if err == nil {
						for imageRows.Next() {
							var url, alt string
							var position int
							if err := imageRows.Scan(&url, &alt, &position); err == nil {
								images = append(images, gin.H{
									"url":      url,
									"alt":      alt,
									"position": position,
								})
							}
						}
						imageRows.Close()
					}

					colors = append(colors, gin.H{
						"id":    colorID,
						"name":  colorName,
						"code":  colorCode,
						"images": images,
					})
				}
			}
			colorRows.Close()
		}

		// Create price range string
		var priceRange string
		if minPrice > 0 && maxPrice > 0 {
			if minPrice == maxPrice {
				priceRange = fmt.Sprintf("%.2f", minPrice)
			} else {
				priceRange = fmt.Sprintf("%.2f - %.2f", minPrice, maxPrice)
			}
		} else {
			priceRange = "0.00 - 0.00"
		}

		product = gin.H{
			"id":                id,
			"brand_id":          brandID,
			"title":             title,
			"description":       description,
			"short_description": shortDescription,
			"model_code":        modelCode,
			"is_active":         isActive,
			"attributes":        attributes,
			"created_at":        createdAt,
			"updated_at":        updatedAt,
			"brand_name":        brandName.String,
			"colors":            colors,
			"min_price":         minPrice,
			"max_price":         maxPrice,
			"price_range":       priceRange,
			"total_stock":       totalStock,
			"variants_count":    variantsCount,
		}
		products = append(products, product)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		WHERE 1=1
	`
	countArgs := []interface{}{}
	countArgIndex := 1

	if status == "active" {
		countQuery += ` AND pm.is_active = true`
	} else if status == "inactive" {
		countQuery += ` AND pm.is_active = false`
	}

	if categoryID != "" {
		countQuery += ` AND pm.id IN (
			SELECT pmc.product_model_id 
			FROM product_model_categories pmc 
			WHERE pmc.category_id = $` + strconv.Itoa(countArgIndex) + `
		)`
		countArgs = append(countArgs, categoryID)
		countArgIndex++
	}

	if brandID != "" {
		countQuery += ` AND pm.brand_id = $` + strconv.Itoa(countArgIndex)
		countArgs = append(countArgs, brandID)
		countArgIndex++
	}

	if search != "" {
		countQuery += ` AND (pm.title ILIKE $` + strconv.Itoa(countArgIndex) + ` OR pm.description ILIKE $` + strconv.Itoa(countArgIndex) + `)`
		searchTerm := "%" + search + "%"
		countArgs = append(countArgs, searchTerm, searchTerm)
	}

	var total int
	err = DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		total = len(products) // Fallback
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetAdminProduct retrieves a single product with all details for admin editing
func GetAdminProduct(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	// Get product model details
	var productModel struct {
		ID               string    `json:"id"`
		BrandID          string    `json:"brand_id"`
		Title            string    `json:"title"`
		Description      string    `json:"description"`
		ShortDescription string    `json:"short_description"`
		ModelCode        string    `json:"model_code"`
		IsActive         bool      `json:"is_active"`
		Attributes       string    `json:"attributes"`
		CreatedAt        time.Time `json:"created_at"`
		UpdatedAt        time.Time `json:"updated_at"`
	}

	productQuery := `SELECT id, brand_id, title, description, short_description, model_code, is_active, attributes, created_at, updated_at 
	                 FROM product_models WHERE id = $1`
	
	err := DB.QueryRow(productQuery, productID).Scan(
		&productModel.ID, &productModel.BrandID, &productModel.Title, &productModel.Description,
		&productModel.ShortDescription, &productModel.ModelCode, &productModel.IsActive,
		&productModel.Attributes, &productModel.CreatedAt, &productModel.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		}
		return
	}

	// Get categories
	var categories []string
	categoryQuery := `SELECT category_id FROM product_model_categories WHERE product_model_id = $1`
	rows, err := DB.Query(categoryQuery, productID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var categoryID string
			if err := rows.Scan(&categoryID); err == nil {
				categories = append(categories, categoryID)
			}
		}
	}

	// Get colors with images
	var colors []gin.H
	colorQuery := `SELECT id, color_name, color_code FROM product_colors WHERE product_model_id = $1 ORDER BY created_at`
	rows, err = DB.Query(colorQuery, productID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var colorID, colorName, colorCode string
			if err := rows.Scan(&colorID, &colorName, &colorCode); err == nil {
				// Get images for this color
				var images []gin.H
				imageQuery := `SELECT url, alt, position FROM product_images WHERE product_color_id = $1 ORDER BY position`
				imageRows, err := DB.Query(imageQuery, colorID)
				if err == nil {
					for imageRows.Next() {
						var url, alt string
						var position int
						if err := imageRows.Scan(&url, &alt, &position); err == nil {
							images = append(images, gin.H{
								"url":      url,
								"alt":      alt,
								"position": position,
							})
						}
					}
					imageRows.Close()
				}

				colors = append(colors, gin.H{
					"id":    colorID,
					"name":  colorName,
					"code":  colorCode,
					"images": images,
				})
			}
		}
	}

	// Get SKUs
	var skus []gin.H
	skuQuery := `SELECT s.id, s.sku_code, s.ean, s.size, s.size_normalized, s.attributes, s.created_at,
	                    pc.color_name, p.list_price, p.sale_price, i.available
	             FROM skus s
	             JOIN product_colors pc ON s.product_color_id = pc.id
	             LEFT JOIN prices p ON s.id = p.sku_id
	             LEFT JOIN inventory i ON s.id = i.sku_id
	             WHERE s.product_model_id = $1
	             ORDER BY pc.color_name, s.size`
	
	rows, err = DB.Query(skuQuery, productID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var skuID, skuCode, ean, size, sizeNormalized, attributes, colorName string
			var listPrice, salePrice sql.NullFloat64
			var available sql.NullInt64
			var createdAt time.Time
			
			if err := rows.Scan(&skuID, &skuCode, &ean, &size, &sizeNormalized, &attributes, &createdAt,
				&colorName, &listPrice, &salePrice, &available); err == nil {
				
				sku := gin.H{
					"id":         skuID,
					"sku_code":   skuCode,
					"ean":        ean,
					"size":       size,
					"color_name": colorName,
					"price":      listPrice.Float64,
					"sale_price": salePrice.Float64,
					"inventory":  available.Int64,
				}
				skus = append(skus, sku)
			}
		}
	}

	// Parse attributes
	var attributesMap map[string]interface{}
	if productModel.Attributes != "" {
		json.Unmarshal([]byte(productModel.Attributes), &attributesMap)
	} else {
		attributesMap = make(map[string]interface{})
	}

	// Return complete product data
	c.JSON(http.StatusOK, gin.H{
		"id":                productModel.ID,
		"brand_id":          productModel.BrandID,
		"title":             productModel.Title,
		"description":       productModel.Description,
		"short_description": productModel.ShortDescription,
		"model_code":        productModel.ModelCode,
		"is_active":         productModel.IsActive,
		"attributes":        attributesMap,
		"categories":        categories,
		"colors":            colors,
		"skus":              skus,
		"created_at":        productModel.CreatedAt,
		"updated_at":        productModel.UpdatedAt,
	})
}
// CreateProduct creates a new product with all necessary information
func CreateProduct(c *gin.Context) {
	var req struct {
		BrandID          string   `json:"brand_id" binding:"required"`
		Title            string   `json:"title" binding:"required"`
		Description      string   `json:"description"`
		ShortDescription string   `json:"short_description"`
		Categories       []string `json:"categories"`
		Colors           []struct {
			Name  string `json:"name" binding:"required"`
			Code  string `json:"code"`
			Images []struct {
				URL     string `json:"url" binding:"required"`
				Alt     string `json:"alt"`
				Position int    `json:"position"`
			} `json:"images"`
		} `json:"colors" binding:"required"`
		SKUs []struct {
			ColorName string  `json:"color_name" binding:"required"`
			Size      string  `json:"size"`
			Price     float64 `json:"price" binding:"required"`
			SalePrice float64 `json:"sale_price"`
			Inventory int     `json:"inventory" binding:"required"`
		} `json:"skus" binding:"required"`
		Attributes map[string]interface{} `json:"attributes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start transaction
	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Create product model
	productModelID := uuid.New()
	now := time.Now()
	
	// Get brand name for code generation
	var brandName string
	err = DB.QueryRow("SELECT name FROM brands WHERE id = $1", req.BrandID).Scan(&brandName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid brand ID"})
		return
	}
	
	// Generate unique product code
	productCode := generateProductCode(req.Title, brandName)
	
	// Convert attributes to JSON
	attributesJSON := "{}"
	if req.Attributes != nil && len(req.Attributes) > 0 {
		// Convert map to JSON string properly
		jsonBytes, err := json.Marshal(req.Attributes)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attributes format"})
			return
		}
		attributesJSON = string(jsonBytes)
	}

	productQuery := `INSERT INTO product_models (id, brand_id, title, description, short_description, model_code, is_active, attributes, created_at, updated_at) 
	                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	
	_, err = tx.Exec(productQuery, productModelID, req.BrandID, req.Title, req.Description, 
	                 req.ShortDescription, productCode, true, attributesJSON, now, now)
	if err != nil {
		fmt.Printf("Error creating product model: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product model", "details": err.Error()})
		return
	}

	// Link to categories
	for _, categoryID := range req.Categories {
		categoryQuery := `INSERT INTO product_model_categories (product_model_id, category_id) VALUES ($1, $2)`
		_, err = tx.Exec(categoryQuery, productModelID, categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link product to category"})
			return
		}
	}

	// Create product colors and images
	colorMap := make(map[string]uuid.UUID)
	for _, color := range req.Colors {
		colorID := uuid.New()
		colorQuery := `INSERT INTO product_colors (id, product_model_id, color_name, color_code, created_at) 
		               VALUES ($1, $2, $3, $4, $5)`
		
		_, err = tx.Exec(colorQuery, colorID, productModelID, color.Name, color.Code, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product color"})
			return
		}

		colorMap[color.Name] = colorID

		// Create images for this color
		for _, image := range color.Images {
			imageID := uuid.New()
			imageQuery := `INSERT INTO product_images (id, product_model_id, product_color_id, url, alt, position, created_at) 
			               VALUES ($1, $2, $3, $4, $5, $6, $7)`
			
			_, err = tx.Exec(imageQuery, imageID, productModelID, colorID, image.URL, image.Alt, image.Position, now)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product image"})
				return
			}
		}
	}

		// Create SKUs
		for _, sku := range req.SKUs {
			skuID := uuid.New()
			
			// Handle empty size for products like sunglasses
			skuSize := sku.Size
			if skuSize == "" {
				skuSize = "One Size"
			}
			
			skuCode := generateSKUCode(productCode, sku.ColorName, skuSize)
			eanCode := generateEANCode(productCode, sku.ColorName, skuSize)
			
			// Get color ID
			colorID, exists := colorMap[sku.ColorName]
			if !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Color not found: " + sku.ColorName})
				return
			}

			skuQuery := `INSERT INTO skus (id, product_model_id, product_color_id, sku_code, ean, size, size_normalized, attributes, created_at) 
			             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
			
			sizeNormalized := normalizeSize(skuSize)
			_, err = tx.Exec(skuQuery, skuID, productModelID, colorID, skuCode, eanCode, skuSize, sizeNormalized, "{}", now)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create SKU"})
				return
			}

		// Create inventory record
		inventoryQuery := `INSERT INTO inventory (sku_id, available, reserved, updated_at) VALUES ($1, $2, $3, $4)`
		_, err = tx.Exec(inventoryQuery, skuID, sku.Inventory, 0, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inventory"})
			return
		}

		// Create price record
		priceID := uuid.New()
		priceQuery := `INSERT INTO prices (id, sku_id, currency, list_price, sale_price, created_at) 
		               VALUES ($1, $2, $3, $4, $5, $6)`
		
		salePrice := sku.SalePrice
		if salePrice == 0 {
			salePrice = sku.Price
		}
		
		_, err = tx.Exec(priceQuery, priceID, skuID, "MRO", sku.Price, salePrice, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create price"})
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"product_id": productModelID,
		"product_code": productCode,
	})
}

// UpdateProduct updates an existing product
func UpdateProduct(c *gin.Context) {
	productID := c.Param("id")
	
	var req struct {
		Title            string  `json:"title"`
		Description      string  `json:"description"`
		ShortDescription string  `json:"short_description"`
		IsActive         *bool   `json:"is_active,omitempty"`
		Attributes       map[string]interface{} `json:"attributes"`
		Colors           []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Code  string `json:"code"`
			Images []struct {
				URL     string `json:"url"`
				Alt     string `json:"alt"`
				Position int    `json:"position"`
			} `json:"images"`
		} `json:"colors"`
		SKUs []struct {
			ID        string  `json:"id"`
			ColorName string  `json:"color_name"`
			Size      string  `json:"size"`
			Price     float64 `json:"price"`
			SalePrice float64 `json:"sale_price"`
			Inventory int     `json:"inventory"`
		} `json:"skus"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start transaction
	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Build dynamic update query for product_models
	query := "UPDATE product_models SET "
	args := []interface{}{}
	argIndex := 1

	if req.Title != "" {
		query += "title = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, req.Title)
		argIndex++
	}

	if req.Description != "" {
		query += "description = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, req.Description)
		argIndex++
	}

	if req.ShortDescription != "" {
		query += "short_description = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, req.ShortDescription)
		argIndex++
	}

	if req.IsActive != nil {
		query += "is_active = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.IsActive)
		argIndex++
	}

	if req.Attributes != nil {
		attributesJSON, err := json.Marshal(req.Attributes)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attributes format"})
			return
		}
		query += "attributes = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, string(attributesJSON))
		argIndex++
	}

	query += "updated_at = now() WHERE id = $" + strconv.Itoa(argIndex)
	args = append(args, productID)

	_, err = tx.Exec(query, args...)
	if err != nil {
		fmt.Printf("Update product error: %v\n", err)
		fmt.Printf("Query: %s\n", query)
		fmt.Printf("Args: %v\n", args)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product", "details": err.Error()})
		return
	}

	// Handle colors and images if provided
	if len(req.Colors) > 0 {
		fmt.Printf("Processing %d colors for product %s\n", len(req.Colors), productID)
		
		// First, get all existing colors for this product
		var existingColors []string
		existingQuery := `SELECT id FROM product_colors WHERE product_model_id = $1`
		existingRows, err := tx.Query(existingQuery, productID)
		if err == nil {
			for existingRows.Next() {
				var existingID string
				if err := existingRows.Scan(&existingID); err == nil {
					existingColors = append(existingColors, existingID)
				}
			}
			existingRows.Close()
		}
		fmt.Printf("Found %d existing colors for product %s\n", len(existingColors), productID)
		
		// Track which colors we're keeping
		colorsToKeep := make(map[string]bool)
		
		for _, colorData := range req.Colors {
			var colorID string
			
			// Check if color ID is valid UUID and exists
			if colorData.ID != "" && len(colorData.ID) == 36 {
				// Check if this color exists for this product
				err = tx.QueryRow("SELECT id FROM product_colors WHERE id = $1 AND product_model_id = $2", 
					colorData.ID, productID).Scan(&colorID)
				
				if err == nil {
					// Color exists, update it
					_, err = tx.Exec("UPDATE product_colors SET color_name = $1, color_code = $2 WHERE id = $3",
						colorData.Name, colorData.Code, colorID)
					if err != nil {
						fmt.Printf("Error updating color: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update color"})
						return
					}
					colorsToKeep[colorID] = true
					fmt.Printf("Updated existing color: %s\n", colorID)
				} else {
					// Color ID doesn't exist, create new one
					colorID = uuid.New().String()
					_, err = tx.Exec("INSERT INTO product_colors (id, product_model_id, color_name, color_code, created_at) VALUES ($1, $2, $3, $4, now())",
						colorID, productID, colorData.Name, colorData.Code)
					if err != nil {
						fmt.Printf("Error creating color: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create color"})
						return
					}
					colorsToKeep[colorID] = true
					fmt.Printf("Created new color (ID not found): %s for product %s\n", colorID, productID)
				}
			} else {
				// No valid ID, create new color
				colorID = uuid.New().String()
				_, err = tx.Exec("INSERT INTO product_colors (id, product_model_id, color_name, color_code, created_at) VALUES ($1, $2, $3, $4, now())",
					colorID, productID, colorData.Name, colorData.Code)
				if err != nil {
					fmt.Printf("Error creating color: %v\n", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create color"})
					return
				}
				colorsToKeep[colorID] = true
				fmt.Printf("Created new color: %s for product %s\n", colorID, productID)
			}

			// Handle images for this color
			if len(colorData.Images) > 0 {
				fmt.Printf("Processing %d images for color %s\n", len(colorData.Images), colorID)
				
				// Delete existing images for this color
				_, err = tx.Exec("DELETE FROM product_images WHERE product_color_id = $1", colorID)
				if err != nil {
					fmt.Printf("Error deleting existing images: %v\n", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing images"})
					return
				}

				// Insert new images with proper positioning
				for index, img := range colorData.Images {
					imageID := uuid.New().String()
					position := img.Position
					if position == 0 {
						position = index + 1 // Use array index + 1 if position not specified
					}
					
					_, err = tx.Exec("INSERT INTO product_images (id, product_model_id, product_color_id, url, alt, position, created_at) VALUES ($1, $2, $3, $4, $5, $6, now())",
						imageID, productID, colorID, img.URL, img.Alt, position)
					if err != nil {
						fmt.Printf("Error inserting image: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert image"})
						return
					}
					fmt.Printf("Inserted image %d: %s (position: %d)\n", index+1, img.URL, position)
				}
			} else {
				fmt.Printf("No images provided for color %s\n", colorID)
			}
		}
		
		// Delete colors that are no longer needed
		for _, existingColorID := range existingColors {
			if !colorsToKeep[existingColorID] {
				fmt.Printf("Deleting unused color: %s\n", existingColorID)
				// Delete images first
				_, err = tx.Exec("DELETE FROM product_images WHERE product_color_id = $1", existingColorID)
				if err != nil {
					fmt.Printf("Error deleting images for color %s: %v\n", existingColorID, err)
				}
				// Delete color
				_, err = tx.Exec("DELETE FROM product_colors WHERE id = $1", existingColorID)
				if err != nil {
					fmt.Printf("Error deleting color %s: %v\n", existingColorID, err)
				}
			}
		}
	}

	// Handle SKUs if provided
	if len(req.SKUs) > 0 {
		fmt.Printf("Processing %d SKUs for product %s\n", len(req.SKUs), productID)
		
		// Get existing SKUs for this product
		var existingSKUs []string
		existingSKUQuery := `SELECT id FROM skus WHERE product_model_id = $1`
		existingSKURows, err := tx.Query(existingSKUQuery, productID)
		if err == nil {
			for existingSKURows.Next() {
				var existingSKUID string
				if err := existingSKURows.Scan(&existingSKUID); err == nil {
					existingSKUs = append(existingSKUs, existingSKUID)
				}
			}
			existingSKURows.Close()
		}
		fmt.Printf("Found %d existing SKUs for product %s\n", len(existingSKUs), productID)
		
		// Track which SKUs we're keeping
		skusToKeep := make(map[string]bool)
		
		for _, skuData := range req.SKUs {
			var skuID string
			
			// Check if SKU ID is valid UUID and exists
			if skuData.ID != "" && len(skuData.ID) == 36 {
				// Check if this SKU exists for this product
				err = tx.QueryRow("SELECT id FROM skus WHERE id = $1 AND product_model_id = $2", 
					skuData.ID, productID).Scan(&skuID)
				
				if err == nil {
					// SKU exists, update it
					// Handle empty size for products like sunglasses
					skuSize := skuData.Size
					if skuSize == "" {
						skuSize = "One Size"
					}
					
					// Update SKU
					_, err = tx.Exec("UPDATE skus SET size = $1, size_normalized = $2 WHERE id = $3",
						skuSize, normalizeSize(skuSize), skuID)
					if err != nil {
						fmt.Printf("Error updating SKU: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update SKU"})
						return
					}
					
					// Update inventory
					_, err = tx.Exec("UPDATE inventory SET available = $1 WHERE sku_id = $2",
						skuData.Inventory, skuID)
					if err != nil {
						fmt.Printf("Error updating inventory: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update inventory"})
						return
					}
					
					// Update price
					salePrice := skuData.SalePrice
					if salePrice == 0 {
						salePrice = skuData.Price
					}
					_, err = tx.Exec("UPDATE prices SET list_price = $1, sale_price = $2 WHERE sku_id = $3",
						skuData.Price, salePrice, skuID)
					if err != nil {
						fmt.Printf("Error updating price: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update price"})
						return
					}
					
					skusToKeep[skuID] = true
					fmt.Printf("Updated existing SKU: %s\n", skuID)
				} else {
					// SKU ID doesn't exist, create new one
					skuID = uuid.New().String()
					
					// Find color ID by name
					var colorID string
					err = tx.QueryRow("SELECT id FROM product_colors WHERE product_model_id = $1 AND color_name = $2", 
						productID, skuData.ColorName).Scan(&colorID)
					if err != nil {
						fmt.Printf("Error finding color %s for SKU: %v\n", skuData.ColorName, err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Color not found: " + skuData.ColorName})
						return
					}
					
				// Handle empty size for products like sunglasses
				skuSize := skuData.Size
				if skuSize == "" {
					skuSize = "One Size"
				}
				
				// Generate SKU code and EAN
				skuCode := generateSKUCode("PROD", skuData.ColorName, skuSize)
				eanCode := generateEANCode("PROD", skuData.ColorName, skuSize)
				
				// Create SKU
				_, err = tx.Exec("INSERT INTO skus (id, product_model_id, product_color_id, sku_code, ean, size, size_normalized, attributes, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())",
					skuID, productID, colorID, skuCode, eanCode, skuSize, normalizeSize(skuSize), "{}")
					if err != nil {
						fmt.Printf("Error creating SKU: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create SKU"})
						return
					}
					
					// Create inventory
					_, err = tx.Exec("INSERT INTO inventory (sku_id, available, reserved, updated_at) VALUES ($1, $2, $3, now())",
						skuID, skuData.Inventory, 0)
					if err != nil {
						fmt.Printf("Error creating inventory: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inventory"})
						return
					}
					
					// Create price
					priceID := uuid.New().String()
					salePrice := skuData.SalePrice
					if salePrice == 0 {
						salePrice = skuData.Price
					}
					_, err = tx.Exec("INSERT INTO prices (id, sku_id, currency, list_price, sale_price, created_at) VALUES ($1, $2, $3, $4, $5, now())",
						priceID, skuID, "MRO", skuData.Price, salePrice)
					if err != nil {
						fmt.Printf("Error creating price: %v\n", err)
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create price"})
						return
					}
					
					skusToKeep[skuID] = true
					fmt.Printf("Created new SKU: %s for color %s\n", skuID, skuData.ColorName)
				}
			} else {
				// No valid ID, create new SKU
				skuID = uuid.New().String()
				
				// Find color ID by name
				var colorID string
				err = tx.QueryRow("SELECT id FROM product_colors WHERE product_model_id = $1 AND color_name = $2", 
					productID, skuData.ColorName).Scan(&colorID)
				if err != nil {
					fmt.Printf("Error finding color %s for SKU: %v\n", skuData.ColorName, err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Color not found: " + skuData.ColorName})
					return
				}
				
				// Handle empty size for products like sunglasses
				skuSize := skuData.Size
				if skuSize == "" {
					skuSize = "One Size"
				}
				
				// Generate SKU code and EAN
				skuCode := generateSKUCode("PROD", skuData.ColorName, skuSize)
				eanCode := generateEANCode("PROD", skuData.ColorName, skuSize)
				
				// Create SKU
				_, err = tx.Exec("INSERT INTO skus (id, product_model_id, product_color_id, sku_code, ean, size, size_normalized, attributes, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())",
					skuID, productID, colorID, skuCode, eanCode, skuSize, normalizeSize(skuSize), "{}")
				if err != nil {
					fmt.Printf("Error creating SKU: %v\n", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create SKU"})
					return
				}
				
				// Create inventory
				_, err = tx.Exec("INSERT INTO inventory (sku_id, available, reserved, updated_at) VALUES ($1, $2, $3, now())",
					skuID, skuData.Inventory, 0)
				if err != nil {
					fmt.Printf("Error creating inventory: %v\n", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inventory"})
					return
				}
				
				// Create price
				priceID := uuid.New().String()
				salePrice := skuData.SalePrice
				if salePrice == 0 {
					salePrice = skuData.Price
				}
				_, err = tx.Exec("INSERT INTO prices (id, sku_id, currency, list_price, sale_price, created_at) VALUES ($1, $2, $3, $4, $5, now())",
					priceID, skuID, "MRO", skuData.Price, salePrice)
				if err != nil {
					fmt.Printf("Error creating price: %v\n", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create price"})
					return
				}
				
				skusToKeep[skuID] = true
				fmt.Printf("Created new SKU: %s for color %s\n", skuID, skuData.ColorName)
			}
		}
		
		// Delete SKUs that are no longer needed
		for _, existingSKUID := range existingSKUs {
			if !skusToKeep[existingSKUID] {
				fmt.Printf("Deleting unused SKU: %s\n", existingSKUID)
				// Delete price first
				_, err = tx.Exec("DELETE FROM prices WHERE sku_id = $1", existingSKUID)
				if err != nil {
					fmt.Printf("Error deleting price for SKU %s: %v\n", existingSKUID, err)
				}
				// Delete inventory
				_, err = tx.Exec("DELETE FROM inventory WHERE sku_id = $1", existingSKUID)
				if err != nil {
					fmt.Printf("Error deleting inventory for SKU %s: %v\n", existingSKUID, err)
				}
				// Delete SKU
				_, err = tx.Exec("DELETE FROM skus WHERE id = $1", existingSKUID)
				if err != nil {
					fmt.Printf("Error deleting SKU %s: %v\n", existingSKUID, err)
				}
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	fmt.Printf("Product %s updated successfully with images and SKUs\n", productID)
	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
}

// DeleteProduct deletes a product and all related data
func DeleteProduct(c *gin.Context) {
	productID := c.Param("id")

	// Start transaction
	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Delete in reverse order of dependencies
	queries := []string{
		"DELETE FROM prices WHERE sku_id IN (SELECT id FROM skus WHERE product_model_id = $1)",
		"DELETE FROM inventory WHERE sku_id IN (SELECT id FROM skus WHERE product_model_id = $1)",
		"DELETE FROM product_images WHERE product_model_id = $1",
		"DELETE FROM skus WHERE product_model_id = $1",
		"DELETE FROM product_colors WHERE product_model_id = $1",
		"DELETE FROM product_model_categories WHERE product_model_id = $1",
		"DELETE FROM product_models WHERE id = $1",
	}

	for _, query := range queries {
		_, err = tx.Exec(query, productID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product data"})
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// UploadImage handles image uploads to Cloudinary
func UploadImage(c *gin.Context) {
	fmt.Printf("Upload request received\n")
	
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Printf("No file provided: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	folder := c.DefaultPostForm("folder", "uploads")
	fmt.Printf("Uploading file: %s to folder: %s\n", file.Filename, folder)
	
	// Check if Cloudinary is initialized
	if services.Cloudinary == nil {
		fmt.Printf("ERROR: Cloudinary not initialized, returning mock URL\n")
		fmt.Printf("Cloudinary service is nil - check initialization logs\n")
		// Return mock URL if Cloudinary is not configured
		mockURL := fmt.Sprintf("https://via.placeholder.com/400x300/cccccc/666666?text=%s", file.Filename)
		c.JSON(http.StatusOK, gin.H{
			"url":       mockURL,
			"public_id": fmt.Sprintf("mock_%d_%s", time.Now().Unix(), file.Filename),
			"width":     400,
			"height":    300,
		})
		return
	}
	
	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Read file data
	fileData := make([]byte, file.Size)
	_, err = src.Read(fileData)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	fmt.Printf("File data read successfully, size: %d bytes\n", len(fileData))

	// Upload to Cloudinary
	uploadResult, err := services.Cloudinary.UploadImageFromBytes(fileData, file.Filename, folder)
	if err != nil {
		fmt.Printf("Cloudinary upload failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to Cloudinary", "details": err.Error()})
		return
	}

	fmt.Printf("Cloudinary upload successful: %s\n", uploadResult.URL)

	c.JSON(http.StatusOK, gin.H{
		"url":       uploadResult.URL,
		"public_id": uploadResult.PublicID,
		"width":     uploadResult.Width,
		"height":    uploadResult.Height,
	})
}

// Helper functions
func generateProductCode(title, brandName string) string {
	// Generate a professional product code: BRAND-TITLE-NUMBER
	cleanBrand := strings.ReplaceAll(strings.ToUpper(brandName), " ", "")
	cleanBrand = strings.ReplaceAll(cleanBrand, "[^A-Z0-9]", "")
	if len(cleanBrand) > 6 {
		cleanBrand = cleanBrand[:6]
	}
	
	cleanTitle := strings.ReplaceAll(strings.ToUpper(title), " ", "-")
	cleanTitle = strings.ReplaceAll(cleanTitle, "[^A-Z0-9-]", "")
	if len(cleanTitle) > 12 {
		cleanTitle = cleanTitle[:12]
	}
	
	// Add a random number to ensure uniqueness
	randomNum := time.Now().UnixNano() % 10000
	return fmt.Sprintf("%s-%s-%04d", cleanBrand, cleanTitle, randomNum)
}

func generateSKUCode(productCode, colorName, size string) string {
	// Generate SKU code: PRODUCT-COLOR-SIZE
	cleanColor := strings.ReplaceAll(strings.ToUpper(colorName), " ", "")
	cleanSize := strings.ReplaceAll(strings.ToUpper(size), " ", "")
	return fmt.Sprintf("%s-%s-%s", productCode, cleanColor[:4], cleanSize)
}

func generateEANCode(productCode, colorName, size string) string {
	// Generate a proper 13-digit EAN code
	// Use a combination of product info to create unique identifier
	
	// Create a unique string from product info
	baseString := fmt.Sprintf("%s%s%s", productCode, colorName, size)
	
	// Convert to numeric representation (only digits)
	var numericString strings.Builder
	for _, char := range baseString {
		if char >= '0' && char <= '9' {
			numericString.WriteRune(char)
		} else if char >= 'A' && char <= 'Z' {
			// Convert letters to numbers (A=1, B=2, etc.)
			numericString.WriteString(fmt.Sprintf("%d", int(char-'A')+1))
		} else if char >= 'a' && char <= 'z' {
			// Convert lowercase letters to numbers (a=1, b=2, etc.)
			numericString.WriteString(fmt.Sprintf("%d", int(char-'a')+1))
		}
	}
	
	// Take first 12 digits and pad with zeros if needed
	base := numericString.String()
	if len(base) > 12 {
		base = base[:12]
	} else {
		// Pad with zeros to make it exactly 12 digits
		for len(base) < 12 {
			base = "0" + base
		}
	}
	
	// Calculate EAN-13 check digit (proper algorithm)
	sum := 0
	for i, digit := range base {
		num := int(digit - '0')
		if i%2 == 0 {
			sum += num
		} else {
			sum += num * 3
		}
	}
	checkDigit := (10 - (sum % 10)) % 10
	
	// Return 13-digit EAN code
	return fmt.Sprintf("%s%d", base, checkDigit)
}

func normalizeSize(size string) string {
	// Normalize size for sorting (e.g., "42 EU" -> "42")
	parts := strings.Fields(size)
	if len(parts) > 0 {
		return parts[0]
	}
	return size
}
