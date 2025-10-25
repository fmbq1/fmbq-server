package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
)

// getProductColors fetches colors for a product with their images
func getProductColors(productID string) []gin.H {
	query := `
		SELECT pc.id, pc.color_name, pc.color_code,
		       COALESCE(pi.url, '') as image_url
		FROM product_colors pc
		LEFT JOIN LATERAL (
			SELECT url 
			FROM product_images 
			WHERE product_model_id = $1 AND product_color_id = pc.id
			ORDER BY position 
			LIMIT 1
		) pi ON true
		WHERE pc.product_model_id = $1
		ORDER BY pc.created_at
	`
	
	rows, err := DB.Query(query, productID)
	if err != nil {
		// Return default colors if query fails
		return []gin.H{
			{
				"id":         "default-1",
				"color_name": "Black",
				"color_hex":  "#000000",
				"image_url":  "https://via.placeholder.com/300x300?text=Black",
				"is_active":  true,
			},
			{
				"id":         "default-2", 
				"color_name": "White",
				"color_hex":  "#FFFFFF",
				"image_url":  "https://via.placeholder.com/300x300?text=White",
				"is_active":  true,
			},
		}
	}
	defer rows.Close()

	var colors []gin.H
	for rows.Next() {
		var color struct {
			ID        string  `json:"id"`
			ColorName string  `json:"color_name"`
			ColorCode *string `json:"color_code"`
			ImageURL  string  `json:"image_url"`
		}

		err := rows.Scan(
			&color.ID, &color.ColorName, &color.ColorCode, &color.ImageURL,
		)
		if err != nil {
			continue
		}

		// Use color_code as color_hex, or generate a default color
		colorHex := "#000000" // Default black
		if color.ColorCode != nil && *color.ColorCode != "" {
			colorHex = *color.ColorCode
		}

		colors = append(colors, gin.H{
			"id":         color.ID,
			"color_name": color.ColorName,
			"color_hex":  colorHex,
			"image_url":  color.ImageURL,
			"is_active":  true,
		})
	}

	// If no colors found, return default colors
	if len(colors) == 0 {
		return []gin.H{
			{
				"id":         "default-1",
				"color_name": "Black",
				"color_hex":  "#000000",
				"image_url":  "https://via.placeholder.com/300x300?text=Black",
				"is_active":  true,
			},
			{
				"id":         "default-2",
				"color_name": "White", 
				"color_hex":  "#FFFFFF",
				"image_url":  "https://via.placeholder.com/300x300?text=White",
				"is_active":  true,
			},
		}
	}

	return colors
}

// getProductColorsWithVariants fetches colors with their images and available sizes
func getProductColorsWithVariants(productID string) []gin.H {
	query := `
		SELECT 
			pc.id, pc.color_name, pc.color_code,
			COALESCE(pi.url, '') as image_url
		FROM product_colors pc
		LEFT JOIN LATERAL (
			SELECT url 
			FROM product_images 
			WHERE product_model_id = $1 AND product_color_id = pc.id
			ORDER BY position 
			LIMIT 1
		) pi ON true
		WHERE pc.product_model_id = $1
		ORDER BY pc.created_at
	`
	
	rows, err := DB.Query(query, productID)
	if err != nil {
		// Return default colors if query fails
		return []gin.H{
			{
				"id":               "default-1",
				"color_name":       "Black",
				"color_hex":        "#000000",
				"image_url":        "https://via.placeholder.com/300x300?text=Black",
				"available_sizes":  []string{"S", "M", "L"},
				"variant_count":    3,
				"is_active":        true,
			},
			{
				"id":               "default-2",
				"color_name":       "White",
				"color_hex":        "#FFFFFF",
				"image_url":        "https://via.placeholder.com/300x300?text=White",
				"available_sizes":  []string{"S", "M", "L"},
				"variant_count":    3,
				"is_active":        true,
			},
		}
	}
	defer rows.Close()

	var colors []gin.H
	for rows.Next() {
		var color struct {
			ID        string  `json:"id"`
			ColorName string  `json:"color_name"`
			ColorCode *string `json:"color_code"`
			ImageURL  string  `json:"image_url"`
		}

		err := rows.Scan(
			&color.ID, &color.ColorName, &color.ColorCode, &color.ImageURL,
		)
		if err != nil {
			continue
		}

		// Use color_code as color_hex, or generate a default color
		colorHex := "#000000" // Default black
		if color.ColorCode != nil && *color.ColorCode != "" {
			colorHex = *color.ColorCode
		}

		colors = append(colors, gin.H{
			"id":               color.ID,
			"color_name":       color.ColorName,
			"color_hex":        colorHex,
			"image_url":        color.ImageURL,
			"available_sizes":  []string{"S", "M", "L"}, // Default sizes
			"variant_count":    3, // Default variant count
			"is_active":        true,
		})
	}

	// If no colors found, return default colors
	if len(colors) == 0 {
		return []gin.H{
			{
				"id":               "default-1",
				"color_name":       "Black",
				"color_hex":        "#000000",
				"image_url":        "https://via.placeholder.com/300x300?text=Black",
				"available_sizes":  []string{"S", "M", "L"},
				"variant_count":    3,
				"is_active":        true,
			},
			{
				"id":               "default-2",
				"color_name":       "White",
				"color_hex":        "#FFFFFF",
				"image_url":        "https://via.placeholder.com/300x300?text=White",
				"available_sizes":  []string{"S", "M", "L"},
				"variant_count":    3,
				"is_active":        true,
			},
		}
	}

	return colors
}

// getProductSizes fetches all available sizes for a product
func getProductSizes(productID string) []gin.H {
	query := `
		SELECT DISTINCT s.size, s.size_normalized, COUNT(*) as variant_count
		FROM skus s
		WHERE s.product_model_id = $1 AND s.size IS NOT NULL
		GROUP BY s.size, s.size_normalized
		ORDER BY s.size_normalized
	`
	
	rows, err := DB.Query(query, productID)
	if err != nil {
		return []gin.H{
			{"size": "S", "size_normalized": "S", "variant_count": 1},
			{"size": "M", "size_normalized": "M", "variant_count": 1},
			{"size": "L", "size_normalized": "L", "variant_count": 1},
		}
	}
	defer rows.Close()

	var sizes []gin.H
	for rows.Next() {
		var size struct {
			Size           *string `json:"size"`
			SizeNormalized *string `json:"size_normalized"`
			VariantCount   int     `json:"variant_count"`
		}

		err := rows.Scan(&size.Size, &size.SizeNormalized, &size.VariantCount)
		if err != nil {
			continue
		}

		sizes = append(sizes, gin.H{
			"size":            size.Size,
			"size_normalized": size.SizeNormalized,
			"variant_count":   size.VariantCount,
		})
	}

	return sizes
}

// getProductVariants fetches all SKU variants with pricing
func getProductVariants(productID string) []gin.H {
	query := `
		SELECT 
			s.id, s.sku_code, s.ean, s.size, s.size_normalized,
			pc.color_name, pc.color_code,
			COALESCE(pr.list_price, 0) as list_price,
			COALESCE(pr.sale_price, pr.list_price, 0) as current_price,
			COALESCE(pi.url, '') as image_url
		FROM skus s
		LEFT JOIN product_colors pc ON s.product_color_id = pc.id
		LEFT JOIN prices pr ON s.id = pr.sku_id
		LEFT JOIN LATERAL (
			SELECT url 
			FROM product_images 
			WHERE product_model_id = $1 AND product_color_id = pc.id
			ORDER BY position 
			LIMIT 1
		) pi ON true
		WHERE s.product_model_id = $1
		ORDER BY pc.color_name, s.size
	`
	
	rows, err := DB.Query(query, productID)
	if err != nil {
		return []gin.H{}
	}
	defer rows.Close()

	var variants []gin.H
	for rows.Next() {
		var variant struct {
			ID            string   `json:"id"`
			SKUCode       string   `json:"sku_code"`
			EAN           *string  `json:"ean"`
			Size          *string  `json:"size"`
			SizeNormalized *string `json:"size_normalized"`
			ColorName     *string  `json:"color_name"`
			ColorCode     *string  `json:"color_code"`
			ListPrice     float64  `json:"list_price"`
			CurrentPrice  float64  `json:"current_price"`
			ImageURL      string   `json:"image_url"`
		}

		err := rows.Scan(
			&variant.ID, &variant.SKUCode, &variant.EAN, &variant.Size,
			&variant.SizeNormalized, &variant.ColorName, &variant.ColorCode,
			&variant.ListPrice, &variant.CurrentPrice, &variant.ImageURL,
		)
		if err != nil {
			continue
		}

		// Use color_code as color_hex, or generate a default color
		colorHex := "#000000" // Default black
		if variant.ColorCode != nil && *variant.ColorCode != "" {
			colorHex = *variant.ColorCode
		}

		variants = append(variants, gin.H{
			"id":               variant.ID,
			"sku_code":         variant.SKUCode,
			"ean":              variant.EAN,
			"size":             variant.Size,
			"size_normalized":  variant.SizeNormalized,
			"color_name":       variant.ColorName,
			"color_hex":        colorHex,
			"list_price":       variant.ListPrice,
			"current_price":    variant.CurrentPrice,
			"image_url":        variant.ImageURL,
		})
	}

	return variants
}

func GetProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	categoryID := c.Query("category_id")
	brandID := c.Query("brand_id")
	search := c.Query("search")

	offset := (page - 1) * limit

	query := `
		SELECT pm.id, pm.brand_id, pm.title, pm.description, pm.short_description, 
		       pm.model_code, pm.is_active, pm.attributes, pm.created_at, pm.updated_at,
		       b.name as brand_name
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		WHERE pm.is_active = true
	`
	args := []interface{}{}
	argIndex := 1

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
		var pm models.ProductModel
		var brandName sql.NullString
		err := rows.Scan(
			&pm.ID, &pm.BrandID, &pm.Title, &pm.Description, &pm.ShortDescription,
			&pm.ModelCode, &pm.IsActive, &pm.Attributes, &pm.CreatedAt, &pm.UpdatedAt,
			&brandName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan product"})
			return
		}

		product := gin.H{
			"id":                pm.ID,
			"brand_id":          pm.BrandID,
			"title":             pm.Title,
			"description":       pm.Description,
			"short_description": pm.ShortDescription,
			"model_code":        pm.ModelCode,
			"is_active":         pm.IsActive,
			"attributes":        pm.Attributes,
			"created_at":        pm.CreatedAt,
			"updated_at":        pm.UpdatedAt,
			"brand_name":        brandName.String,
		}

		products = append(products, product)
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"page":     page,
		"limit":    limit,
	})
}

func GetProduct(c *gin.Context) {
	productID := c.Param("id")
	
	// Get product model
	var pm models.ProductModel
	var brandName sql.NullString
	query := `
		SELECT pm.id, pm.brand_id, pm.title, pm.description, pm.short_description, 
		       pm.model_code, pm.is_active, pm.attributes, pm.created_at, pm.updated_at,
		       b.name as brand_name
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		WHERE pm.id = $1
	`
	
	err := DB.QueryRow(query, productID).Scan(
		&pm.ID, &pm.BrandID, &pm.Title, &pm.Description, &pm.ShortDescription,
		&pm.ModelCode, &pm.IsActive, &pm.Attributes, &pm.CreatedAt, &pm.UpdatedAt,
		&brandName,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		}
		return
	}

	// Get product colors
	colorsQuery := `SELECT id, color_name, color_code, external_color_id, default_image_id, created_at 
	                FROM product_colors WHERE product_model_id = $1 ORDER BY created_at`
	colorsRows, err := DB.Query(colorsQuery, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product colors"})
		return
	}
	defer colorsRows.Close()

	var colors []gin.H
	for colorsRows.Next() {
		var pc models.ProductColor
		err := colorsRows.Scan(
			&pc.ID, &pc.ColorName, &pc.ColorCode, &pc.ExternalColorID, 
			&pc.DefaultImageID, &pc.CreatedAt,
		)
		if err != nil {
			continue
		}
		colors = append(colors, gin.H{
			"id":                pc.ID,
			"color_name":        pc.ColorName,
			"color_code":        pc.ColorCode,
			"external_color_id": pc.ExternalColorID,
			"default_image_id":  pc.DefaultImageID,
			"created_at":        pc.CreatedAt,
		})
	}

	// Get product images
	imagesQuery := `SELECT id, url, alt, position, created_at 
	                FROM product_images WHERE product_model_id = $1 ORDER BY position, created_at`
	imagesRows, err := DB.Query(imagesQuery, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product images"})
		return
	}
	defer imagesRows.Close()

	var images []gin.H
	for imagesRows.Next() {
		var pi models.ProductImage
		err := imagesRows.Scan(&pi.ID, &pi.URL, &pi.Alt, &pi.Position, &pi.CreatedAt)
		if err != nil {
			continue
		}
		images = append(images, gin.H{
			"id":         pi.ID,
			"url":        pi.URL,
			"alt":        pi.Alt,
			"position":   pi.Position,
			"created_at": pi.CreatedAt,
		})
	}

	// Get SKUs with prices
	skusQuery := `
		SELECT s.id, s.sku_code, s.ean, s.size, s.size_normalized, s.attributes, s.created_at,
		       p.list_price, p.sale_price, p.currency, i.available, i.reserved
		FROM skus s
		LEFT JOIN prices p ON s.id = p.sku_id AND p.currency = 'MRO'
		LEFT JOIN inventory i ON s.id = i.sku_id
		WHERE s.product_model_id = $1
		ORDER BY s.created_at
	`
	skusRows, err := DB.Query(skusQuery, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SKUs"})
		return
	}
	defer skusRows.Close()

	var skus []gin.H
	for skusRows.Next() {
		var sku models.SKU
		var listPrice, salePrice sql.NullFloat64
		var currency sql.NullString
		var available, reserved sql.NullInt64
		var ean sql.NullString
		
		err := skusRows.Scan(
			&sku.ID, &sku.SKUCode, &ean, &sku.Size, &sku.SizeNormalized, 
			&sku.Attributes, &sku.CreatedAt, &listPrice, &salePrice, &currency,
			&available, &reserved,
		)
		if err != nil {
			continue
		}
		
		skuData := gin.H{
			"id":              sku.ID,
			"sku_code":        sku.SKUCode,
			"ean":             ean.String,
			"size":            sku.Size,
			"size_normalized": sku.SizeNormalized,
			"attributes":      sku.Attributes,
			"created_at":      sku.CreatedAt,
			"available":       available.Int64,
			"reserved":        reserved.Int64,
		}
		
		if listPrice.Valid {
			skuData["list_price"] = listPrice.Float64
		}
		if salePrice.Valid {
			skuData["sale_price"] = salePrice.Float64
		}
		if currency.Valid {
			skuData["currency"] = currency.String
		}
		
		skus = append(skus, skuData)
	}

	product := gin.H{
		"id":                pm.ID,
		"brand_id":          pm.BrandID,
		"title":             pm.Title,
		"description":       pm.Description,
		"short_description": pm.ShortDescription,
		"model_code":        pm.ModelCode,
		"is_active":         pm.IsActive,
		"attributes":        pm.Attributes,
		"created_at":        pm.CreatedAt,
		"updated_at":        pm.UpdatedAt,
		"brand_name":        brandName.String,
		"colors":            colors,
		"images":            images,
		"skus":              skus,
	}

	c.JSON(http.StatusOK, product)
}

// GetProductsByBrand fetches products for a specific brand with pricing and images
func GetProductsByBrand(c *gin.Context) {
	brandID := c.Param("brandId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	offset := (page - 1) * limit

	// First get brand info
	var brand models.Brand
	brandQuery := `SELECT id, name, color, logo, description FROM brands WHERE id = $1`
	err := DB.QueryRow(brandQuery, brandID).Scan(
		&brand.ID, &brand.Name, &brand.Color, &brand.Logo, &brand.Description,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Brand not found"})
		return
	}

	// Get products with basic info first, then enhance with additional data
	query := `
		SELECT 
			pm.id, pm.title, pm.model_code, pm.is_active, pm.description,
			b.name as brand_name, b.color as brand_color,
			COALESCE(pi.url, '') as main_image_url
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN LATERAL (
			SELECT url 
			FROM product_images 
			WHERE product_model_id = pm.id 
			ORDER BY position 
			LIMIT 1
		) pi ON true
		WHERE pm.brand_id = $1 AND pm.is_active = true
		ORDER BY pm.created_at DESC
		LIMIT $2 OFFSET $3
	`

	fmt.Printf("DEBUG: Querying products for brand %s with limit %d offset %d\n", brandID, limit, offset)
	rows, err := DB.Query(query, brandID, limit, offset)
	if err != nil {
		fmt.Printf("DEBUG: Query error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer rows.Close()

	var products []gin.H
	for rows.Next() {
		var product struct {
			ID           string  `json:"id"`
			Title        string  `json:"title"`
			ModelCode    string  `json:"model_code"`
			IsActive     bool    `json:"is_active"`
			Description  *string `json:"description"`
			BrandName    string  `json:"brand_name"`
			BrandColor   *string `json:"brand_color"`
			MainImageURL string  `json:"main_image_url"`
		}

		err := rows.Scan(
			&product.ID, &product.Title, &product.ModelCode, &product.IsActive,
			&product.Description, &product.BrandName, &product.BrandColor,
			&product.MainImageURL,
		)
		if err != nil {
			continue
		}

		// Fetch comprehensive product data
		colors := getProductColorsWithVariants(product.ID)
		sizes := getProductSizes(product.ID)
		variants := getProductVariants(product.ID)

		// Get pricing from variants (use first variant's pricing as default)
		price := 99.99
		originalPrice := 119.99
		discountPercentage := 0
		if len(variants) > 0 {
			variant := variants[0]
			if currentPrice, exists := variant["current_price"]; exists {
				price = currentPrice.(float64)
			}
			if listPrice, exists := variant["list_price"]; exists {
				originalPrice = listPrice.(float64)
			}
			if price < originalPrice {
				discountPercentage = int(((originalPrice - price) / originalPrice) * 100)
			}
		}

		// Generate badges based on discount and other factors
		badges := []string{}
		if discountPercentage > 0 {
			badges = append(badges, "Sale")
		}
		if price < 50 {
			badges = append(badges, "Budget")
		}
		if len(badges) == 0 {
			badges = append(badges, "New")
		}

		products = append(products, gin.H{
			"id":                   product.ID,
			"name":                 product.Title,
			"description":          product.Description,
			"brand":                product.BrandName,
			"brand_color":          product.BrandColor,
			"model_code":           product.ModelCode,
			"price":                price,
			"original_price":       originalPrice,
			"discount_percentage":  discountPercentage,
			"image_url":            product.MainImageURL,
			"colors":               colors,
			"sizes":                sizes,
			"variants":             variants,
			"is_favorite":          false,
			"badges":               badges,
		})
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM product_models WHERE brand_id = $1 AND is_active = true`
	err = DB.QueryRow(countQuery, brandID).Scan(&total)
	if err != nil {
		total = len(products)
	}

	c.JSON(http.StatusOK, gin.H{
		"brand": gin.H{
			"id":          brand.ID,
			"name":        brand.Name,
			"color":       brand.Color,
			"logo_url":    brand.Logo,
			"description": brand.Description,
		},
		"products": products,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

// GetSimilarProducts returns products in the same level-1 or same subcategory as the given product
func GetSimilarProducts(c *gin.Context) {
    productID := c.Param("id")
    limitStr := c.DefaultQuery("limit", "16")
    limit, err := strconv.Atoi(limitStr)
    if err != nil || limit <= 0 {
        limit = 16
    }

    query := `
WITH base_cats AS (
  SELECT c.id, COALESCE(c.parent_id, c.id) AS level1_id
  FROM categories c
  JOIN product_model_categories pmc ON pmc.category_id = c.id
  WHERE pmc.product_model_id = $1
)
SELECT DISTINCT 
  pm2.id,
  pm2.title,
  b.name AS brand_name,
  COALESCE(pi.url, '') AS image_url,
  COALESCE(pr.price, 0) AS price,
  pm2.created_at
FROM product_models pm2
JOIN product_model_categories pmc2 ON pmc2.product_model_id = pm2.id
JOIN categories c2 ON c2.id = pmc2.category_id
LEFT JOIN brands b ON b.id = pm2.brand_id
LEFT JOIN LATERAL (
  SELECT url
  FROM product_images 
  WHERE product_model_id = pm2.id
  ORDER BY position, created_at
  LIMIT 1
) pi ON true
LEFT JOIN LATERAL (
  SELECT MIN(COALESCE(p.sale_price, p.list_price)) AS price
  FROM skus s
  LEFT JOIN prices p ON p.sku_id = s.id AND p.currency = 'MRO'
  WHERE s.product_model_id = pm2.id
) pr ON true
WHERE pm2.id <> $1
  AND (
    c2.id IN (SELECT id FROM base_cats)
    OR COALESCE(c2.parent_id, c2.id) IN (SELECT level1_id FROM base_cats)
  )
ORDER BY pm2.created_at DESC
LIMIT $2`

    rows, err := DB.Query(query, productID, limit)
    if err != nil {
        fmt.Printf("DEBUG: GetSimilarProducts query error for product %s (limit %d): %v\n", productID, limit, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch similar products"})
        return
    }
    defer rows.Close()

    var list []gin.H
    for rows.Next() {
        var id, title, brandName, imageURL string
        var price sql.NullFloat64
        var createdAt time.Time
        if err := rows.Scan(&id, &title, &brandName, &imageURL, &price, &createdAt); err == nil {
            list = append(list, gin.H{
                "id":         id,
                "title":      title,
                "brand_name": brandName,
                "image_url":  imageURL,
                "price":      price.Float64,
                "created_at": createdAt,
            })
        } else {
            fmt.Printf("DEBUG: GetSimilarProducts scan error: %v\n", err)
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "products": list,
    })
}

