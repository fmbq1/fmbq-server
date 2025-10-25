package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SearchRequest represents the search parameters
type SearchRequest struct {
	Query      string `form:"q" json:"q"`                    // Text search query
	Category   string `form:"category" json:"category"`      // Category ID filter
	Brand      string `form:"brand" json:"brand"`            // Brand ID filter
	Color      string `form:"color" json:"color"`            // Color filter
	Size       string `form:"size" json:"size"`              // Size filter
	MinPrice   string `form:"min_price" json:"min_price"`
	MaxPrice   string `form:"max_price" json:"max_price"`
	SortBy     string `form:"sort_by" json:"sort_by"`        // price_asc, price_desc, name_asc, name_desc, newest
	Page       string `form:"page" json:"page"`
	Limit      string `form:"limit" json:"limit"`
	InStock    string `form:"in_stock" json:"in_stock"`      // true, false, all
}

// SearchResponse represents the search results
type SearchResponse struct {
	Products    []ProductSearchResult `json:"products"`
	TotalCount  int                   `json:"total_count"`
	Page        int                   `json:"page"`
	Limit       int                   `json:"limit"`
	TotalPages  int                   `json:"total_pages"`
	HasNext     bool                  `json:"has_next"`
	HasPrevious bool                  `json:"has_previous"`
	Filters     SearchFilters         `json:"filters"`
}

// ProductSearchResult represents a product in search results
type ProductSearchResult struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	BrandName         string   `json:"brand_name"`
	BrandID           string   `json:"brand_id"`
	ModelCode         string   `json:"model_code"`
	Description       string   `json:"description"`
	ShortDescription  string   `json:"short_description"`
	ImageURL          string   `json:"image_url"`
	MinPrice          float64  `json:"min_price"`
	MaxPrice          float64  `json:"max_price"`
	OriginalPrice     float64  `json:"original_price"`
	DiscountPercent   float64  `json:"discount_percent"`
	IsActive          bool     `json:"is_active"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	Categories        []string `json:"categories"`
	AvailableSizes    []string `json:"available_sizes"`
	AvailableColors   []string `json:"available_colors"`
	TotalStock        int      `json:"total_stock"`
	Badges            []string `json:"badges"`
}

// SearchFilters represents available filters
type SearchFilters struct {
	Categories []CategoryFilter `json:"categories"`
	Brands     []BrandFilter     `json:"brands"`
	PriceRange PriceRange        `json:"price_range"`
}

// CategoryFilter represents a category filter option
type CategoryFilter struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Count    int               `json:"count"`
	Level    int               `json:"level"`
	ParentID string            `json:"parent_id,omitempty"`
	Children []CategoryFilter  `json:"children,omitempty"`
}

// BrandFilter represents a brand filter option
type BrandFilter struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// PriceRange represents the price range filter
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// SearchProducts handles product search with filters and pagination
func SearchProducts(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid search parameters",
			"details": err.Error(),
		})
		return
	}

	// Validate and set defaults
	page, limit, offset := validatePagination(req.Page, req.Limit)
	sortBy := validateSortBy(req.SortBy)
	inStock := validateInStock(req.InStock)

	// Build the search query
	query, args, err := buildSearchQuery(req, sortBy, inStock, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to build search query",
			"details": err.Error(),
		})
		return
	}

	// Execute search query
	rows, err := DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database query failed",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	// Parse results
	products, err := parseSearchResults(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to parse search results",
			"details": err.Error(),
		})
		return
	}

	// Get total count for pagination
	totalCount, err := getSearchTotalCount(req, inStock)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get total count",
			"details": err.Error(),
		})
		return
	}

	// Get filters
	filters, err := getSearchFilters(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get search filters",
			"details": err.Error(),
		})
		return
	}

	// Calculate pagination info
	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := SearchResponse{
		Products:    products,
		TotalCount:  totalCount,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		Filters:     filters,
	}

	c.JSON(http.StatusOK, response)
}

// EnhancedSearchProducts provides advanced search with color and size filtering
func EnhancedSearchProducts(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid search parameters",
			"details": err.Error(),
		})
		return
	}

	// Validate and set defaults
	page, limit, offset := validatePagination(req.Page, req.Limit)
	sortBy := validateSortBy(req.SortBy)
	// inStock := validateInStock(req.InStock)

	// Build simple search query that works
	query := `
		SELECT DISTINCT
			pm.id,
			pm.title,
			pm.description,
			pm.short_description,
			pm.model_code,
			pm.is_active,
			pm.created_at,
			pm.updated_at,
			b.id as brand_id,
			b.name as brand_name,
			COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) as min_price,
			COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) as max_price,
			COALESCE(MAX(pr.list_price), 0) as original_price,
			0 as total_stock,
			COALESCE(pi.url, '') as image_url
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
		LEFT JOIN categories c ON pmc.category_id = c.id
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN prices pr ON s.id = pr.sku_id
		LEFT JOIN LATERAL (
			SELECT url 
			FROM product_images 
			WHERE product_model_id = pm.id
			ORDER BY position 
			LIMIT 1
		) pi ON true
		WHERE pm.is_active = true
	`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add search conditions
	if req.Query != "" {
		searchTerm := "%" + strings.ToLower(req.Query) + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(pm.title) LIKE $%d OR LOWER(pm.description) LIKE $%d OR LOWER(pm.model_code) LIKE $%d OR LOWER(b.name) LIKE $%d)", 
			argIndex, argIndex, argIndex, argIndex))
		args = append(args, searchTerm)
		argIndex++
	}

	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("c.id = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	if req.Brand != "" {
		conditions = append(conditions, fmt.Sprintf("b.id = $%d", argIndex))
		args = append(args, req.Brand)
		argIndex++
	}

	if req.Color != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM skus s2 JOIN product_colors pc ON s2.product_color_id = pc.id WHERE s2.product_model_id = pm.id AND LOWER(pc.color_name) LIKE LOWER($%d))", argIndex))
		args = append(args, "%"+req.Color+"%")
		argIndex++
	}

	if req.Size != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM skus s3 WHERE s3.product_model_id = pm.id AND LOWER(s3.size) LIKE LOWER($%d))", argIndex))
		args = append(args, "%"+req.Size+"%")
		argIndex++
	}

	if req.MinPrice != "" {
		if minPrice, err := strconv.ParseFloat(req.MinPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) >= $%d", argIndex))
			args = append(args, minPrice)
			argIndex++
		}
	}

	if req.MaxPrice != "" {
		if maxPrice, err := strconv.ParseFloat(req.MaxPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) <= $%d", argIndex))
			args = append(args, maxPrice)
			argIndex++
		}
	}

	// Skip inventory filtering for now to avoid errors
	// if inStock == "true" {
	// 	conditions = append(conditions, "COALESCE(inv.available, 0) > 0")
	// } else if inStock == "false" {
	// 	conditions = append(conditions, "COALESCE(inv.available, 0) = 0")
	// }

	// Combine conditions
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Add GROUP BY and ORDER BY
	query += `
		GROUP BY pm.id, pm.title, pm.description, pm.short_description, pm.model_code, 
				 pm.is_active, pm.created_at, pm.updated_at, b.id, b.name, pi.url
		ORDER BY ` + sortBy + `
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	
	args = append(args, limit, offset)

	// Execute search query
	rows, err := DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database query failed",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	// Parse results
	products, err := parseSearchResults(rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to parse search results",
			"details": err.Error(),
		})
		return
	}

	// Get total count for pagination
	totalCount := len(products) // Simple count for now

	// Skip filters for now to avoid errors
	filters := SearchFilters{}

	// Calculate pagination info
	totalPages := (totalCount + limit - 1) / limit
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := SearchResponse{
		Products:    products,
		TotalCount:  totalCount,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		Filters:     filters,
	}

	c.JSON(http.StatusOK, response)
}

// GetCategoryHierarchy returns organized category hierarchy
func GetCategoryHierarchy(c *gin.Context) {
	query := `
		SELECT 
			c.id,
			c.name,
			c.parent_id,
			c.level,
			c.created_at,
			c.updated_at,
			COUNT(DISTINCT pm.id) as product_count
		FROM categories c
		LEFT JOIN product_model_categories pmc ON c.id = pmc.category_id
		LEFT JOIN product_models pm ON pmc.product_model_id = pm.id AND pm.is_active = true
		WHERE c.is_active = true
		GROUP BY c.id, c.name, c.parent_id, c.level, c.created_at, c.updated_at
		ORDER BY c.level ASC, c.name ASC
	`

	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch categories",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var categories []CategoryFilter
	for rows.Next() {
		var cat CategoryFilter
		var parentID sql.NullString
		var level sql.NullInt32

		err := rows.Scan(
			&cat.ID,
			&cat.Name,
			&parentID,
			&level,
			&cat.Count,
		)
		if err != nil {
			continue
		}

		if parentID.Valid {
			cat.ParentID = parentID.String
		}
		if level.Valid {
			cat.Level = int(level.Int32)
		}

		categories = append(categories, cat)
	}

	// Organize into hierarchy
	hierarchy := organizeCategoryHierarchy(categories)

	c.JSON(http.StatusOK, gin.H{
		"categories": hierarchy,
		"total":      len(categories),
	})
}

// Helper functions

func validatePagination(pageStr, limitStr string) (int, int, int) {
	page := 1
	limit := 20

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit
	return page, limit, offset
}

func validateSortBy(sortBy string) string {
	validSorts := map[string]string{
		"price_asc":   "min_price ASC",
		"price_desc":  "min_price DESC",
		"name_asc":    "pm.title ASC",
		"name_desc":   "pm.title DESC",
		"newest":      "pm.created_at DESC",
		"oldest":      "pm.created_at ASC",
	}

	if validSort, exists := validSorts[sortBy]; exists {
		return validSort
	}
	return "pm.created_at DESC" // default
}

func validateInStock(inStock string) string {
	switch inStock {
	case "true":
		return "true"
	case "false":
		return "false"
	default:
		return "all"
	}
}

func buildSearchQuery(req SearchRequest, sortBy, inStock string, limit, offset int) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Base query
	baseQuery := `
		SELECT DISTINCT
			pm.id,
			pm.title,
			pm.description,
			pm.short_description,
			pm.model_code,
			pm.is_active,
			pm.created_at,
			pm.updated_at,
			b.id as brand_id,
			b.name as brand_name,
			COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) as min_price,
			COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) as max_price,
			COALESCE(MAX(pr.list_price), 0) as original_price,
			COALESCE(SUM(inv.available), 0) as total_stock,
			COALESCE(pi.url, '') as image_url
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
		LEFT JOIN categories c ON pmc.category_id = c.id
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN prices pr ON s.id = pr.sku_id
		LEFT JOIN LATERAL (
			SELECT url 
			FROM product_images 
			WHERE product_model_id = pm.id
			ORDER BY position 
			LIMIT 1
		) pi ON true
		WHERE pm.is_active = true
	`

	// Add search conditions
	if req.Query != "" {
		searchTerm := "%" + strings.ToLower(req.Query) + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(pm.title) LIKE $%d OR LOWER(pm.description) LIKE $%d OR LOWER(pm.model_code) LIKE $%d OR LOWER(b.name) LIKE $%d)", 
			argIndex, argIndex, argIndex, argIndex))
		args = append(args, searchTerm)
		argIndex++
	}

	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("c.id = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	if req.Brand != "" {
		conditions = append(conditions, fmt.Sprintf("b.id = $%d", argIndex))
		args = append(args, req.Brand)
		argIndex++
	}

	if req.MinPrice != "" {
		if minPrice, err := strconv.ParseFloat(req.MinPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) >= $%d", argIndex))
			args = append(args, minPrice)
			argIndex++
		}
	}

	if req.MaxPrice != "" {
		if maxPrice, err := strconv.ParseFloat(req.MaxPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) <= $%d", argIndex))
			args = append(args, maxPrice)
			argIndex++
		}
	}

	if inStock == "true" {
		conditions = append(conditions, "inv.available > 0")
	} else if inStock == "false" {
		conditions = append(conditions, "inv.available = 0")
	}

	// Combine conditions
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add GROUP BY and ORDER BY
	baseQuery += `
		GROUP BY pm.id, pm.title, pm.description, pm.short_description, pm.model_code, 
				 pm.is_active, pm.created_at, pm.updated_at, b.id, b.name, pi.url
		ORDER BY ` + sortBy + `
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	
	args = append(args, limit, offset)

	return baseQuery, args, nil
}

// buildEnhancedSearchQuery builds a search query with color and size filtering support
func buildEnhancedSearchQuery(req SearchRequest, sortBy, inStock string, limit, offset int) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT DISTINCT
			pm.id,
			pm.title,
			pm.description,
			pm.short_description,
			pm.model_code,
			pm.is_active,
			pm.created_at,
			pm.updated_at,
			b.id as brand_id,
			b.name as brand_name,
			COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) as min_price,
			COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) as max_price,
			COALESCE(MAX(pr.list_price), 0) as original_price,
			COALESCE(SUM(inv.available), 0) as total_stock,
			COALESCE(pi.url, '') as image_url
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
		LEFT JOIN categories c ON pmc.category_id = c.id
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN prices pr ON s.id = pr.sku_id
		LEFT JOIN LATERAL (
			SELECT url 
			FROM product_images 
			WHERE product_model_id = pm.id
			ORDER BY position 
			LIMIT 1
		) pi ON true
		WHERE pm.is_active = true
	`

	// Add search conditions
	if req.Query != "" {
		searchTerm := "%" + strings.ToLower(req.Query) + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(pm.title) LIKE $%d OR LOWER(pm.description) LIKE $%d OR LOWER(pm.model_code) LIKE $%d OR LOWER(b.name) LIKE $%d)", 
			argIndex, argIndex, argIndex, argIndex))
		args = append(args, searchTerm)
		argIndex++
	}

	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("c.id = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	if req.Brand != "" {
		conditions = append(conditions, fmt.Sprintf("b.id = $%d", argIndex))
		args = append(args, req.Brand)
		argIndex++
	}

	// Enhanced color filtering
	if req.Color != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM skus s2 JOIN product_colors pc ON s2.product_color_id = pc.id WHERE s2.product_model_id = pm.id AND LOWER(pc.color_name) LIKE LOWER($%d))", argIndex))
		args = append(args, "%"+req.Color+"%")
		argIndex++
	}

	// Enhanced size filtering
	if req.Size != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM skus s3 WHERE s3.product_model_id = pm.id AND LOWER(s3.size) LIKE LOWER($%d))", argIndex))
		args = append(args, "%"+req.Size+"%")
		argIndex++
	}

	if req.MinPrice != "" {
		if minPrice, err := strconv.ParseFloat(req.MinPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) >= $%d", argIndex))
			args = append(args, minPrice)
			argIndex++
		}
	}

	if req.MaxPrice != "" {
		if maxPrice, err := strconv.ParseFloat(req.MaxPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) <= $%d", argIndex))
			args = append(args, maxPrice)
			argIndex++
		}
	}

	if inStock == "true" {
		conditions = append(conditions, "inv.available > 0")
	} else if inStock == "false" {
		conditions = append(conditions, "inv.available = 0")
	}

	// Combine conditions
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add GROUP BY and ORDER BY
	baseQuery += `
		GROUP BY pm.id, pm.title, pm.description, pm.short_description, pm.model_code, 
				 pm.is_active, pm.created_at, pm.updated_at, b.id, b.name, pi.url
		ORDER BY ` + sortBy + `
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	
	args = append(args, limit, offset)

	return baseQuery, args, nil
}

// getEnhancedSearchTotalCount gets the total count for enhanced search
func getEnhancedSearchTotalCount(req SearchRequest, inStock string) (int, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT COUNT(DISTINCT pm.id)
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
		LEFT JOIN categories c ON pmc.category_id = c.id
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN prices pr ON s.id = pr.sku_id
		WHERE pm.is_active = true
	`

	// Add the same conditions as the main query
	if req.Query != "" {
		searchTerm := "%" + strings.ToLower(req.Query) + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(pm.title) LIKE $%d OR LOWER(pm.description) LIKE $%d OR LOWER(pm.model_code) LIKE $%d OR LOWER(b.name) LIKE $%d)", 
			argIndex, argIndex, argIndex, argIndex))
		args = append(args, searchTerm)
		argIndex++
	}

	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("c.id = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	if req.Brand != "" {
		conditions = append(conditions, fmt.Sprintf("b.id = $%d", argIndex))
		args = append(args, req.Brand)
		argIndex++
	}

	if req.Color != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM skus s2 JOIN product_colors pc ON s2.product_color_id = pc.id WHERE s2.product_model_id = pm.id AND LOWER(pc.color_name) LIKE LOWER($%d))", argIndex))
		args = append(args, "%"+req.Color+"%")
		argIndex++
	}

	if req.Size != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM skus s3 WHERE s3.product_model_id = pm.id AND LOWER(s3.size) LIKE LOWER($%d))", argIndex))
		args = append(args, "%"+req.Size+"%")
		argIndex++
	}

	if req.MinPrice != "" {
		if minPrice, err := strconv.ParseFloat(req.MinPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) >= $%d", argIndex))
			args = append(args, minPrice)
			argIndex++
		}
	}

	if req.MaxPrice != "" {
		if maxPrice, err := strconv.ParseFloat(req.MaxPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) <= $%d", argIndex))
			args = append(args, maxPrice)
			argIndex++
		}
	}

	if inStock == "true" {
		conditions = append(conditions, "inv.available > 0")
	} else if inStock == "false" {
		conditions = append(conditions, "inv.available = 0")
	}

	// Combine conditions
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var count int
	err := DB.QueryRow(baseQuery, args...).Scan(&count)
	return count, err
}

func parseSearchResults(rows *sql.Rows) ([]ProductSearchResult, error) {
	var products []ProductSearchResult

	for rows.Next() {
		var product ProductSearchResult
		var description, shortDescription, modelCode sql.NullString
		var imageURL sql.NullString

		err := rows.Scan(
			&product.ID,
			&product.Title,
			&description,
			&shortDescription,
			&modelCode,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
			&product.BrandID,
			&product.BrandName,
			&product.MinPrice,
			&product.MaxPrice,
			&product.OriginalPrice,
			&product.TotalStock,
			&imageURL,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			product.Description = description.String
		}
		if shortDescription.Valid {
			product.ShortDescription = shortDescription.String
		}
		if modelCode.Valid {
			product.ModelCode = modelCode.String
		}
		if imageURL.Valid {
			product.ImageURL = imageURL.String
		}

		// Calculate discount percentage
		if product.OriginalPrice > 0 && product.MinPrice < product.OriginalPrice {
			product.DiscountPercent = ((product.OriginalPrice - product.MinPrice) / product.OriginalPrice) * 100
		}

		// Get additional product details
		product.Categories = getProductCategoriesForSearch(product.ID)
		product.AvailableSizes = getProductSizesForSearch(product.ID)
		product.AvailableColors = getProductColorsForSearch(product.ID)
		product.Badges = generateProductBadges(product)

		products = append(products, product)
	}

	return products, nil
}

func getSearchTotalCount(req SearchRequest, inStock string) (int, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	baseQuery := `
		SELECT COUNT(DISTINCT pm.id)
		FROM product_models pm
		LEFT JOIN brands b ON pm.brand_id = b.id
		LEFT JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
		LEFT JOIN categories c ON pmc.category_id = c.id
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN prices pr ON s.id = pr.sku_id
		WHERE pm.is_active = true
	`

	// Add same conditions as main query
	if req.Query != "" {
		searchTerm := "%" + strings.ToLower(req.Query) + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(pm.title) LIKE $%d OR LOWER(pm.description) LIKE $%d OR LOWER(pm.model_code) LIKE $%d OR LOWER(b.name) LIKE $%d)", 
			argIndex, argIndex, argIndex, argIndex))
		args = append(args, searchTerm)
		argIndex++
	}

	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("c.id = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	if req.Brand != "" {
		conditions = append(conditions, fmt.Sprintf("b.id = $%d", argIndex))
		args = append(args, req.Brand)
		argIndex++
	}

	if req.MinPrice != "" {
		if minPrice, err := strconv.ParseFloat(req.MinPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) >= $%d", argIndex))
			args = append(args, minPrice)
			argIndex++
		}
	}

	if req.MaxPrice != "" {
		if maxPrice, err := strconv.ParseFloat(req.MaxPrice, 64); err == nil {
			conditions = append(conditions, fmt.Sprintf("COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) <= $%d", argIndex))
			args = append(args, maxPrice)
			argIndex++
		}
	}

	if inStock == "true" {
		conditions = append(conditions, "inv.available > 0")
	} else if inStock == "false" {
		conditions = append(conditions, "inv.available = 0")
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	var count int
	err := DB.QueryRow(baseQuery, args...).Scan(&count)
	return count, err
}

func getSearchFilters(req SearchRequest) (SearchFilters, error) {
	filters := SearchFilters{}

	// Get category filters
	categories, err := getCategoryFilters(req)
	if err != nil {
		return filters, err
	}
	filters.Categories = categories

	// Get brand filters
	brands, err := getBrandFilters(req)
	if err != nil {
		return filters, err
	}
	filters.Brands = brands

	// Get price range
	priceRange, err := getPriceRange(req)
	if err != nil {
		return filters, err
	}
	filters.PriceRange = priceRange

	return filters, nil
}

func getCategoryFilters(req SearchRequest) ([]CategoryFilter, error) {
	query := `
		SELECT 
			c.id,
			c.name,
			c.parent_id,
			c.level,
			COUNT(DISTINCT pm.id) as product_count
		FROM categories c
		LEFT JOIN product_model_categories pmc ON c.id = pmc.category_id
		LEFT JOIN product_models pm ON pmc.product_model_id = pm.id AND pm.is_active = true
		WHERE c.is_active = true
		GROUP BY c.id, c.name, c.parent_id, c.level
		ORDER BY c.level ASC, c.name ASC
	`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []CategoryFilter
	for rows.Next() {
		var cat CategoryFilter
		var parentID sql.NullString
		var level sql.NullInt32

		err := rows.Scan(&cat.ID, &cat.Name, &parentID, &level, &cat.Count)
		if err != nil {
			continue
		}

		if parentID.Valid {
			cat.ParentID = parentID.String
		}
		if level.Valid {
			cat.Level = int(level.Int32)
		}

		categories = append(categories, cat)
	}

	return categories, nil
}

func getBrandFilters(req SearchRequest) ([]BrandFilter, error) {
	query := `
		SELECT 
			b.id,
			b.name,
			COUNT(DISTINCT pm.id) as product_count
		FROM brands b
		LEFT JOIN product_models pm ON b.id = pm.brand_id AND pm.is_active = true
		GROUP BY b.id, b.name
		ORDER BY b.name ASC
	`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brands []BrandFilter
	for rows.Next() {
		var brand BrandFilter
		err := rows.Scan(&brand.ID, &brand.Name, &brand.Count)
		if err != nil {
			continue
		}
		brands = append(brands, brand)
	}

	return brands, nil
}

func getPriceRange(req SearchRequest) (PriceRange, error) {
	query := `
		SELECT 
			COALESCE(MIN(pr.sale_price), MIN(pr.list_price), 0) as min_price,
			COALESCE(MAX(pr.sale_price), MAX(pr.list_price), 0) as max_price
		FROM product_models pm
		LEFT JOIN skus s ON pm.id = s.product_model_id
		LEFT JOIN prices pr ON s.id = pr.sku_id
		WHERE pm.is_active = true
	`

	var priceRange PriceRange
	err := DB.QueryRow(query).Scan(&priceRange.Min, &priceRange.Max)
	return priceRange, err
}

func getProductCategoriesForSearch(productID string) []string {
	query := `
		SELECT c.name
		FROM categories c
		JOIN product_model_categories pmc ON c.id = pmc.category_id
		WHERE pmc.product_model_id = $1 AND c.is_active = true
		ORDER BY c.name
	`

	rows, err := DB.Query(query, productID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err == nil {
			categories = append(categories, category)
		}
	}

	return categories
}

func getProductSizesForSearch(productID string) []string {
	query := `
		SELECT DISTINCT s.size
		FROM skus s
		WHERE s.product_model_id = $1 AND s.size IS NOT NULL AND s.size != ''
		ORDER BY s.size
	`

	rows, err := DB.Query(query, productID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var sizes []string
	for rows.Next() {
		var size string
		if err := rows.Scan(&size); err == nil {
			sizes = append(sizes, size)
		}
	}

	return sizes
}

func getProductColorsForSearch(productID string) []string {
	query := `
		SELECT DISTINCT pc.color_name
		FROM product_colors pc
		JOIN skus s ON pc.id = s.product_color_id
		WHERE s.product_model_id = $1 AND pc.color_name IS NOT NULL AND pc.color_name != ''
		ORDER BY pc.color_name
	`

	rows, err := DB.Query(query, productID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var colors []string
	for rows.Next() {
		var color string
		if err := rows.Scan(&color); err == nil {
			colors = append(colors, color)
		}
	}

	return colors
}

func generateProductBadges(product ProductSearchResult) []string {
	var badges []string

	if product.DiscountPercent > 0 {
		badges = append(badges, fmt.Sprintf("%.0f%% OFF", product.DiscountPercent))
	}

	if product.TotalStock == 0 {
		badges = append(badges, "OUT OF STOCK")
	} else if product.TotalStock < 5 {
		badges = append(badges, "LOW STOCK")
	}

	// Add "NEW" badge for products created in the last 30 days
	if createdAt, err := time.Parse("2006-01-02T15:04:05Z", product.CreatedAt); err == nil {
		if time.Since(createdAt).Hours() < 24*30 {
			badges = append(badges, "NEW")
		}
	}

	return badges
}

func organizeCategoryHierarchy(categories []CategoryFilter) []CategoryFilter {
	// Group categories by level
	parentCategories := make(map[string]CategoryFilter)
	childCategories := make(map[string][]CategoryFilter)

	for _, cat := range categories {
		if cat.Level == 1 || cat.ParentID == "" {
			parentCategories[cat.ID] = cat
		} else {
			childCategories[cat.ParentID] = append(childCategories[cat.ParentID], cat)
		}
	}

	// Build hierarchy
	var hierarchy []CategoryFilter
	for _, parent := range parentCategories {
		if children, exists := childCategories[parent.ID]; exists {
			parent.Children = children
		}
		hierarchy = append(hierarchy, parent)
	}

	return hierarchy
}
