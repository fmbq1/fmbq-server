package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetCategories(c *gin.Context) {
	query := `SELECT id, name, slug, parent_id, external_code, metadata, created_at, updated_at 
	          FROM categories ORDER BY name`
	
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	defer rows.Close()

	var categories []gin.H
	for rows.Next() {
		var cat models.Category
		var parentID sql.NullString
		var externalCode sql.NullString
		err := rows.Scan(
			&cat.ID, &cat.Name, &cat.Slug, &parentID, &externalCode, &cat.Metadata, 
			&cat.CreatedAt, &cat.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Calculate level: 1 if parent_id is null, 2 if parent_id exists
		level := 1
		if parentID.Valid && parentID.String != "" {
			level = 2
		}

		category := gin.H{
			"id":            cat.ID,
			"name":          cat.Name,
			"slug":          cat.Slug,
			"parent_id":     parentID.String,
			"external_code": externalCode.String,
			"metadata":      cat.Metadata,
			"level":         level,
			"created_at":    cat.CreatedAt,
			"updated_at":    cat.UpdatedAt,
		}
		categories = append(categories, category)
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// PublicTopCategories returns ALL categories (both parent and subcategories)
// and is safe for unauthenticated public/mobile clients.
func PublicTopCategories(c *gin.Context) {
    query := `SELECT id, name, slug, parent_id FROM categories WHERE is_active = true ORDER BY level, name`
    
    fmt.Printf("Executing query: %s\n", query)

    rows, err := DB.Query(query)
    if err != nil {
        fmt.Printf("Query error: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
        return
    }
    defer rows.Close()

    type cat struct {
        ID       string  `json:"id"`
        Name     string  `json:"name"`
        Slug     string  `json:"slug"`
        ParentID *string `json:"parent_id"`
        Level    int     `json:"level"`
    }

    list := make([]cat, 0, 32)
    count := 0
    for rows.Next() {
        var item cat
        var parentID sql.NullString
        if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &parentID); err != nil {
            fmt.Printf("Scan error: %v\n", err)
            continue
        }
        if parentID.Valid {
            item.ParentID = &parentID.String
            item.Level = 2 // Subcategory
        } else {
            item.Level = 1 // Parent category
        }
        list = append(list, item)
        count++
        fmt.Printf("Found category: %s (level %d, parent: %v)\n", item.Name, item.Level, item.ParentID)
    }
    
    fmt.Printf("Total categories found: %d\n", count)
    c.JSON(http.StatusOK, gin.H{"categories": list})
}

func GetCategory(c *gin.Context) {
	categoryID := c.Param("id")
	
	var cat models.Category
	var parentID sql.NullString
	var externalCode sql.NullString
	query := `SELECT id, name, slug, parent_id, external_code, metadata, created_at, updated_at 
	          FROM categories WHERE id = $1`
	
	err := DB.QueryRow(query, categoryID).Scan(
		&cat.ID, &cat.Name, &cat.Slug, &parentID, &externalCode, &cat.Metadata, 
		&cat.CreatedAt, &cat.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch category"})
		}
		return
	}

	category := gin.H{
		"id":            cat.ID,
		"name":          cat.Name,
		"slug":          cat.Slug,
		"parent_id":     parentID.String,
		"external_code": externalCode.String,
		"metadata":      cat.Metadata,
		"created_at":    cat.CreatedAt,
		"updated_at":    cat.UpdatedAt,
	}

	c.JSON(http.StatusOK, category)
}

func CreateCategory(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Slug         string `json:"slug" binding:"required"`
		ParentID     string `json:"parent_id,omitempty"`
		ExternalCode string `json:"external_code,omitempty"`
		Metadata     string `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	categoryID := uuid.New()
	var parentUUID *uuid.UUID
	if req.ParentID != "" {
		parent, err := uuid.Parse(req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent_id format"})
			return
		}
		parentUUID = &parent
	}

	var externalCode *string
	if req.ExternalCode != "" {
		externalCode = &req.ExternalCode
	}

	metadata := req.Metadata
	if metadata == "" {
		metadata = "{}"
	}

	query := `INSERT INTO categories (id, name, slug, parent_id, external_code, metadata) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err := DB.Exec(query, categoryID, req.Name, req.Slug, parentUUID, externalCode, metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            categoryID,
		"name":          req.Name,
		"slug":          req.Slug,
		"parent_id":     req.ParentID,
		"external_code": req.ExternalCode,
		"metadata":      metadata,
		"message":       "Category created successfully",
	})
}

func UpdateCategory(c *gin.Context) {
	// Implementation for updating categories
	c.JSON(http.StatusOK, gin.H{"message": "Update category endpoint"})
}

func DeleteCategory(c *gin.Context) {
	// Implementation for deleting categories
	c.JSON(http.StatusOK, gin.H{"message": "Delete category endpoint"})
}
