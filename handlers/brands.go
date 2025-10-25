package handlers

import (
	"database/sql"
	"net/http"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetBrands(c *gin.Context) {
    query := `SELECT id, name, slug, description, external_code, logo, banner, color, parent_category_id, created_at 
              FROM brands ORDER BY name`
	
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch brands"})
		return
	}
	defer rows.Close()

	var brands []gin.H
    for rows.Next() {
		var brand models.Brand
		err := rows.Scan(
            &brand.ID, &brand.Name, &brand.Slug, &brand.Description, 
            &brand.ExternalCode, &brand.Logo, &brand.Banner, &brand.Color, &brand.ParentCategoryID, &brand.CreatedAt,
		)
		if err != nil {
			continue
		}

        brandData := gin.H{
			"id":            brand.ID,
			"name":          brand.Name,
			"slug":          brand.Slug,
			"description":   brand.Description,
			"external_code": brand.ExternalCode,
			"logo":          brand.Logo,
			"banner":        brand.Banner,
			"color":         brand.Color,
            "parent_category_id": brand.ParentCategoryID,
			"created_at":    brand.CreatedAt,
		}
		brands = append(brands, brandData)
	}

	c.JSON(http.StatusOK, gin.H{"brands": brands})
}

func GetBrand(c *gin.Context) {
	brandID := c.Param("id")
	
	var brand models.Brand
    query := `SELECT id, name, slug, description, external_code, logo, banner, color, parent_category_id, created_at 
	          FROM brands WHERE id = $1`
	
	err := DB.QueryRow(query, brandID).Scan(
        &brand.ID, &brand.Name, &brand.Slug, &brand.Description, 
        &brand.ExternalCode, &brand.Logo, &brand.Banner, &brand.Color, &brand.ParentCategoryID, &brand.CreatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Brand not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch brand"})
		}
		return
	}

	brandData := gin.H{
		"id":            brand.ID,
		"name":          brand.Name,
		"slug":          brand.Slug,
		"description":   brand.Description,
		"external_code": brand.ExternalCode,
		"logo":          brand.Logo,
		"banner":        brand.Banner,
		"color":         brand.Color,
        "parent_category_id": brand.ParentCategoryID,
		"created_at":    brand.CreatedAt,
	}

	c.JSON(http.StatusOK, brandData)
}

func CreateBrand(c *gin.Context) {
    var req struct {
		Name         string `json:"name" binding:"required"`
		Slug         string `json:"slug,omitempty"`
		Description  string `json:"description,omitempty"`
		ExternalCode string `json:"external_code,omitempty"`
		Logo         string `json:"logo,omitempty"`
		Banner       string `json:"banner,omitempty"`
		Color        string `json:"color,omitempty"`
        ParentCategoryID string `json:"parent_category_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	brandID := uuid.New()
	
	var slug *string
	if req.Slug != "" {
		slug = &req.Slug
	}
	
	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	
	var externalCode *string
	if req.ExternalCode != "" {
		externalCode = &req.ExternalCode
	}
	
	var logo *string
	if req.Logo != "" {
		logo = &req.Logo
	}
	
	var banner *string
	if req.Banner != "" {
		banner = &req.Banner
	}
	
	var color *string
	if req.Color != "" {
		color = &req.Color
	}

    var parentCategoryID *uuid.UUID
    if req.ParentCategoryID != "" {
        if parsed, err := uuid.Parse(req.ParentCategoryID); err == nil {
            parentCategoryID = &parsed
        }
    }

    query := `INSERT INTO brands (id, name, slug, description, external_code, logo, banner, color, parent_category_id) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
    
    _, err := DB.Exec(query, brandID, req.Name, slug, description, externalCode, logo, banner, color, parentCategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create brand"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            brandID,
		"name":          req.Name,
		"slug":          req.Slug,
		"description":   req.Description,
		"external_code": req.ExternalCode,
		"logo":          req.Logo,
		"banner":        req.Banner,
		"color":         req.Color,
        "parent_category_id": req.ParentCategoryID,
		"message":       "Brand created successfully",
	})
}

func UpdateBrand(c *gin.Context) {
	brandID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(brandID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid brand ID"})
		return
	}

    var req struct {
		Name         string `json:"name" binding:"required"`
		Slug         string `json:"slug,omitempty"`
		Description  string `json:"description,omitempty"`
		ExternalCode string `json:"external_code,omitempty"`
		Logo         string `json:"logo,omitempty"`
		Banner       string `json:"banner,omitempty"`
		Color        string `json:"color,omitempty"`
        ParentCategoryID string `json:"parent_category_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if brand exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM brands WHERE id = $1)`
	err = DB.QueryRow(checkQuery, brandID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Brand not found"})
		return
	}

	// Prepare values for update
	var slug *string
	if req.Slug != "" {
		slug = &req.Slug
	}
	
	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	
	var externalCode *string
	if req.ExternalCode != "" {
		externalCode = &req.ExternalCode
	}
	
	var logo *string
	if req.Logo != "" {
		logo = &req.Logo
	}
	
	var banner *string
	if req.Banner != "" {
		banner = &req.Banner
	}
	
	var color *string
	if req.Color != "" {
		color = &req.Color
	}

    var parentCategoryID *uuid.UUID
    if req.ParentCategoryID != "" {
        if parsed, err := uuid.Parse(req.ParentCategoryID); err == nil {
            parentCategoryID = &parsed
        }
    }

    // Update brand
    query := `UPDATE brands SET name = $1, slug = $2, description = $3, external_code = $4, logo = $5, banner = $6, color = $7, parent_category_id = $8 WHERE id = $9`
    _, err = DB.Exec(query, req.Name, slug, description, externalCode, logo, banner, color, parentCategoryID, brandID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update brand"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Brand updated successfully",
		"brand": gin.H{
			"id":            brandID,
			"name":          req.Name,
			"slug":          req.Slug,
			"description":   req.Description,
			"external_code": req.ExternalCode,
			"logo":          req.Logo,
			"banner":        req.Banner,
			"color":         req.Color,
            "parent_category_id": req.ParentCategoryID,
		},
	})
}

func DeleteBrand(c *gin.Context) {
	// Implementation for deleting brands
	c.JSON(http.StatusOK, gin.H{"message": "Delete brand endpoint"})
}
