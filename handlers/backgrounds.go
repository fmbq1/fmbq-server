package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"fmbq-server/database"
	"fmbq-server/models"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
)

// GetBackgrounds returns all active backgrounds, optionally filtered by category
func GetBackgrounds(c *gin.Context) {
	db := database.Database

	fmt.Println("Attempting to fetch backgrounds...")

	// Get category parameter from query string
	categoryID := c.Query("category_id")
	fmt.Printf("Category ID filter: %s\n", categoryID)

	// First, check if the backgrounds table exists (and create if not)
	var tableExists bool
	err := db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'backgrounds')").Scan(&tableExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check table existence",
		})
		return
	}

	if !tableExists {
		// Create the table if it doesn't exist
		_, err = db.Exec(models.Background{}.CreateTableSQL())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create backgrounds table",
			})
			return
		}
		fmt.Println("Created backgrounds table")
	} else {
		// Check if new columns exist and add them if they don't
		var actionTypeExists bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'backgrounds' AND column_name = 'action_type')").Scan(&actionTypeExists)
		if err == nil && !actionTypeExists {
			_, err = db.Exec("ALTER TABLE backgrounds ADD COLUMN action_type VARCHAR(50) DEFAULT 'search'")
			if err != nil {
				fmt.Printf("Warning: Failed to add action_type column: %v\n", err)
			} else {
				fmt.Println("Added action_type column to backgrounds table")
			}
		}

		var actionDataExists bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'backgrounds' AND column_name = 'action_data')").Scan(&actionDataExists)
		if err == nil && !actionDataExists {
			_, err = db.Exec("ALTER TABLE backgrounds ADD COLUMN action_data JSONB")
			if err != nil {
				fmt.Printf("Warning: Failed to add action_data column: %v\n", err)
			} else {
				fmt.Println("Added action_data column to backgrounds table")
			}
		}

		// Check if category_id column exists and add it if missing
		var categoryIdExists bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'backgrounds' AND column_name = 'category_id')").Scan(&categoryIdExists)
		if err == nil && !categoryIdExists {
			_, err = db.Exec("ALTER TABLE backgrounds ADD COLUMN category_id UUID REFERENCES categories(id)")
			if err != nil {
				fmt.Printf("Warning: Failed to add category_id column: %v\n", err)
			} else {
				fmt.Println("Added category_id column to backgrounds table")
			}
		}
	}

	// Build query with optional category filter
	var query string
	var args []interface{}

	if categoryID != "" {
		query = `
			SELECT id, title, description, image_url, position, is_active, category_id, action_type, action_data, start_date, end_date, created_at, updated_at
			FROM backgrounds
			WHERE is_active = true
			AND (category_id = $1 OR category_id IS NULL)
			ORDER BY position ASC, created_at DESC
		`
		args = append(args, categoryID)
	} else {
		query = `
			SELECT id, title, description, image_url, position, is_active, category_id, action_type, action_data, start_date, end_date, created_at, updated_at
			FROM backgrounds
			WHERE is_active = true
			ORDER BY position ASC, created_at DESC
		`
	}

	fmt.Println("Executing query:", query)
	if len(args) > 0 {
		fmt.Printf("Query args: %v\n", args)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Printf("Database query error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch backgrounds",
		})
		return
	}
	defer rows.Close()

	var backgrounds []models.Background
	for rows.Next() {
		var bg models.Background
		err := rows.Scan(
			&bg.ID,
			&bg.Title,
			&bg.Description,
			&bg.ImageURL,
			&bg.Position,
			&bg.IsActive,
			&bg.CategoryID,
			&bg.ActionType,
			&bg.ActionData,
			&bg.StartDate,
			&bg.EndDate,
			&bg.CreatedAt,
			&bg.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("Row scan error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to scan background data",
			})
			return
		}
		backgrounds = append(backgrounds, bg)
	}

	fmt.Printf("Successfully fetched %d backgrounds\n", len(backgrounds))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    backgrounds,
	})
}

// GetAdminBackgrounds returns all backgrounds for admin management
func GetAdminBackgrounds(c *gin.Context) {
	db := database.Database

	query := `
		SELECT id, title, description, image_url, position, is_active, category_id, start_date, end_date, created_at, updated_at
		FROM backgrounds 
		ORDER BY position ASC, created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch backgrounds",
		})
		return
	}
	defer rows.Close()

	var backgrounds []models.Background
	for rows.Next() {
		var bg models.Background
		err := rows.Scan(
			&bg.ID,
			&bg.Title,
			&bg.Description,
			&bg.ImageURL,
			&bg.Position,
			&bg.IsActive,
			&bg.CategoryID,
			&bg.ActionType,
			&bg.ActionData,
			&bg.StartDate,
			&bg.EndDate,
			&bg.CreatedAt,
			&bg.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to scan background data",
			})
			return
		}
		backgrounds = append(backgrounds, bg)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    backgrounds,
	})
}

// CreateBackground creates a new background
func CreateBackground(c *gin.Context) {
	var req struct {
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description"`
		ImageURL    string  `json:"image_url" binding:"required"`
		Position    int     `json:"position"`
		IsActive    bool    `json:"is_active"`
		CategoryID  *string `json:"category_id"`
		ActionType  string  `json:"action_type"`
		ActionData  map[string]interface{} `json:"action_data"`
		StartDate   *string `json:"start_date"`
		EndDate     *string `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Parse dates from strings
	var startDate, endDate *time.Time
	if req.StartDate != nil && *req.StartDate != "" {
		parsed, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start_date format",
			})
			return
		}
		startDate = &parsed
	}
	if req.EndDate != nil && *req.EndDate != "" {
		parsed, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid end_date format",
			})
			return
		}
		endDate = &parsed
	}

	// Set default action type if not provided
	if req.ActionType == "" {
		req.ActionType = "search"
	}

	// Marshal action data to JSON
	actionDataJSON := "{}"
	if req.ActionData != nil {
		actionDataBytes, err := json.Marshal(req.ActionData)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid action_data format",
			})
			return
		}
		actionDataJSON = string(actionDataBytes)
	}

	db := database.Database

	// First, check if the backgrounds table exists (and create if not)
	var tableExists bool
	err := db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'backgrounds')").Scan(&tableExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check table existence",
		})
		return
	}

	if !tableExists {
		// Create the table if it doesn't exist
		_, err = db.Exec(models.Background{}.CreateTableSQL())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create backgrounds table",
			})
			return
		}
		fmt.Println("Created backgrounds table")
	} else {
		// Check if new columns exist and add them if they don't
		var actionTypeExists bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'backgrounds' AND column_name = 'action_type')").Scan(&actionTypeExists)
		if err == nil && !actionTypeExists {
			_, err = db.Exec("ALTER TABLE backgrounds ADD COLUMN action_type VARCHAR(50) DEFAULT 'search'")
			if err != nil {
				fmt.Printf("Warning: Failed to add action_type column: %v\n", err)
			} else {
				fmt.Println("Added action_type column to backgrounds table")
			}
		}

		var actionDataExists bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'backgrounds' AND column_name = 'action_data')").Scan(&actionDataExists)
		if err == nil && !actionDataExists {
			_, err = db.Exec("ALTER TABLE backgrounds ADD COLUMN action_data JSONB")
			if err != nil {
				fmt.Printf("Warning: Failed to add action_data column: %v\n", err)
			} else {
				fmt.Println("Added action_data column to backgrounds table")
			}
		}

		// Check if category_id column exists and add it if missing
		var categoryIdExists bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'backgrounds' AND column_name = 'category_id')").Scan(&categoryIdExists)
		if err == nil && !categoryIdExists {
			_, err = db.Exec("ALTER TABLE backgrounds ADD COLUMN category_id UUID REFERENCES categories(id)")
			if err != nil {
				fmt.Printf("Warning: Failed to add category_id column: %v\n", err)
			} else {
				fmt.Println("Added category_id column to backgrounds table")
			}
		}
	}

	query := `
		INSERT INTO backgrounds (title, description, image_url, position, is_active, category_id, action_type, action_data, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	var id string
	var createdAt, updatedAt time.Time
	err = db.QueryRow(query, req.Title, req.Description, req.ImageURL, req.Position, req.IsActive, req.CategoryID, req.ActionType, actionDataJSON, startDate, endDate).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		fmt.Printf("Database error creating background: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create background",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":         id,
			"created_at": createdAt,
			"updated_at": updatedAt,
		},
	})
}

// UpdateBackground updates an existing background
func UpdateBackground(c *gin.Context) {
	backgroundID := c.Param("id")
	if backgroundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Background ID is required",
		})
		return
	}

	var req struct {
		Title       string    `json:"title" binding:"required"`
		Description string    `json:"description"`
		ImageURL    string    `json:"image_url" binding:"required"`
		Position    int       `json:"position"`
		IsActive    bool      `json:"is_active"`
		CategoryID  *string   `json:"category_id"`
		StartDate   *time.Time `json:"start_date"`
		EndDate     *time.Time `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	db := database.Database

	query := `
		UPDATE backgrounds 
		SET title = $1, description = $2, image_url = $3, position = $4, is_active = $5, 
		    category_id = $6, start_date = $7, end_date = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING updated_at
	`

	var updatedAt time.Time
	err := db.QueryRow(query, req.Title, req.Description, req.ImageURL, req.Position, req.IsActive, req.CategoryID, req.StartDate, req.EndDate, backgroundID).Scan(&updatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update background",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"updated_at": updatedAt,
		},
	})
}

// DeleteBackground deletes a background
func DeleteBackground(c *gin.Context) {
	backgroundID := c.Param("id")
	if backgroundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Background ID is required",
		})
		return
	}

	db := database.Database

	query := `DELETE FROM backgrounds WHERE id = $1`
	result, err := db.Exec(query, backgroundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete background",
		})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get affected rows",
		})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Background not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Background deleted successfully",
	})
}

// UploadBackgroundImage handles image upload to Cloudinary
func UploadBackgroundImage(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No image file provided",
		})
		return
	}

	// Read file data
	fileData, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read file",
		})
		return
	}
	defer fileData.Close()

	// Read file bytes
	fileBytes, err := io.ReadAll(fileData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read file data",
		})
		return
	}

	// Upload to Cloudinary using the same service as other uploads
	folder := "backgrounds"
	uploadResult, err := services.Cloudinary.UploadImageFromBytes(fileBytes, file.Filename, folder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to upload image to Cloudinary",
		})
		return
	}

	// Return the uploaded image URL
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"image_url": uploadResult.SecureURL,
			"public_id": uploadResult.PublicID,
			"width":     uploadResult.Width,
			"height":    uploadResult.Height,
		},
	})
}

// GetBackgroundStats returns background statistics for admin dashboard
func GetBackgroundStats(c *gin.Context) {
	db := database.Database

	// Get total backgrounds count
	var totalBackgrounds int
	err := db.QueryRow("SELECT COUNT(*) FROM backgrounds").Scan(&totalBackgrounds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch total backgrounds count",
		})
		return
	}

	// Get active backgrounds count
	var activeBackgrounds int
	err = db.QueryRow("SELECT COUNT(*) FROM backgrounds WHERE is_active = true").Scan(&activeBackgrounds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch active backgrounds count",
		})
		return
	}

	// Get inactive backgrounds count
	var inactiveBackgrounds int
	err = db.QueryRow("SELECT COUNT(*) FROM backgrounds WHERE is_active = false").Scan(&inactiveBackgrounds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch inactive backgrounds count",
		})
		return
	}

	// Get expired backgrounds count (backgrounds with end_date in the past)
	var expiredBackgrounds int
	err = db.QueryRow("SELECT COUNT(*) FROM backgrounds WHERE end_date IS NOT NULL AND end_date < NOW()").Scan(&expiredBackgrounds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch expired backgrounds count",
		})
		return
	}

	stats := gin.H{
		"total_backgrounds":    totalBackgrounds,
		"active_backgrounds":   activeBackgrounds,
		"inactive_backgrounds": inactiveBackgrounds,
		"expired_backgrounds":  expiredBackgrounds,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// CheckBackgroundSchema checks the current schema of the backgrounds table
func CheckBackgroundSchema(c *gin.Context) {
	db := database.Database
	
	// Check if table exists
	var tableExists bool
	err := db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'backgrounds')").Scan(&tableExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check table existence",
		})
		return
	}
	
	if !tableExists {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"table_exists": false,
				"columns": []string{},
			},
		})
		return
	}
	
	// Get column information
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_name = 'backgrounds'
		ORDER BY ordinal_position
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get column information",
		})
		return
	}
	defer rows.Close()
	
	var columns []map[string]interface{}
	for rows.Next() {
		var columnName, dataType, isNullable, columnDefault string
		err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault)
		if err != nil {
			continue
		}
		columns = append(columns, map[string]interface{}{
			"name": columnName,
			"type": dataType,
			"nullable": isNullable == "YES",
			"default": columnDefault,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"table_exists": true,
			"columns": columns,
		},
	})
}
