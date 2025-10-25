package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ProductSearchResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	ModelCode  string `json:"model_code"`
	Brand      *Brand `json:"brand"`
	ImageURL   string `json:"image_url"`
	SKUs       []SKU  `json:"skus"`
}

type Brand struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SKU struct {
	ID          string     `json:"id"`
	SKUCode     string     `json:"sku_code"`
	EAN         string     `json:"ean"`
	Size        string     `json:"size"`
	Price       float64    `json:"price"`
	Available   int        `json:"available"`
	ProductColor *ProductColor `json:"product_color"`
}

type ProductColor struct {
	ID        string `json:"id"`
	ColorName string `json:"color_name"`
	ColorHex  string `json:"color_hex"`
	ImageURL  string `json:"image_url"`
}

func SearchProductByCode(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code parameter is required"})
		return
	}

	// Log the search attempt
	fmt.Printf("Searching for product with code: %s\n", code)

	// Test database connection first
	err := DB.Ping()
	if err != nil {
		fmt.Printf("Database connection error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed", "details": err.Error()})
		return
	}

	// Enhanced search - search model_code, sku_code, and ean with full SKU data
	query := `
		SELECT DISTINCT 
			pm.id, pm.title, pm.model_code, 
			b.id as brand_id, b.name as brand_name,
			pi.image_url
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN LATERAL (
			SELECT image_url FROM product_images 
			WHERE product_model_id = pm.id 
			LIMIT 1
		) pi ON true
		WHERE pm.model_code = $1 
		   OR pm.model_code LIKE '%' || $1 || '%'
		   OR EXISTS (
		       SELECT 1 FROM skus s 
		       WHERE s.product_model_id = pm.id 
		       AND (s.sku_code = $1 OR s.sku_code LIKE '%' || $1 || '%' OR s.ean = $1 OR s.ean LIKE '%' || $1 || '%')
		   )
		ORDER BY pm.id
	`

	fmt.Printf("Executing query: %s with parameter: %s\n", query, code)
	rows, err := DB.Query(query, code)
	if err != nil {
		fmt.Printf("Database query error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed", "details": err.Error()})
		return
	}
	defer rows.Close()

	var product *ProductSearchResponse

	for rows.Next() {
		var (
			productID, title, modelCode string
			brandID, brandName         sql.NullString
			imageURL                   sql.NullString
		)

		err := rows.Scan(
			&productID, &title, &modelCode,
			&brandID, &brandName, &imageURL,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan row"})
			return
		}

		// Initialize product if not done yet
		if product == nil {
			product = &ProductSearchResponse{
				ID:        productID,
				Title:     title,
				ModelCode: modelCode,
				ImageURL:  imageURL.String,
				SKUs:      []SKU{},
			}

			if brandID.Valid {
				product.Brand = &Brand{
					ID:   brandID.String,
					Name: brandName.String,
				}
			}
		}
	}

	if product == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Fetch SKU data for the product
	skuQuery := `
		SELECT 
			s.id, s.sku_code, s.ean, s.size,
			COALESCE(p.sale_price, p.list_price, 0) as price,
			COALESCE(i.available, 0) as available,
			pc.color_name, pc.color_hex
		FROM skus s
		LEFT JOIN prices p ON p.sku_id = s.id
		LEFT JOIN inventory i ON i.sku_id = s.id
		LEFT JOIN product_colors pc ON pc.id = s.product_color_id
		WHERE s.product_model_id = $1
		ORDER BY s.size, pc.color_name
	`

	skuRows, err := DB.Query(skuQuery, product.ID)
	if err != nil {
		fmt.Printf("SKU query error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SKUs"})
		return
	}
	defer skuRows.Close()

	var skus []SKU
	for skuRows.Next() {
		var (
			skuID, skuCode, ean, size string
			price, available          float64
			colorName, colorHex       sql.NullString
		)

		err := skuRows.Scan(
			&skuID, &skuCode, &ean, &size,
			&price, &available,
			&colorName, &colorHex,
		)
		if err != nil {
			fmt.Printf("SKU scan error: %v\n", err)
			continue
		}

		sku := SKU{
			ID:      skuID,
			SKUCode: skuCode,
			EAN:     ean,
			Size:    size,
			Price:   price,
			Available: int(available),
		}

		if colorName.Valid {
			sku.ProductColor = &ProductColor{
				ColorName: colorName.String,
				ColorHex: colorHex.String,
			}
		}

		skus = append(skus, sku)
	}

	product.SKUs = skus

	c.JSON(http.StatusOK, product)
}
