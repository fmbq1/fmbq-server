package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fmbq-server/database"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
)

// GetBanners returns all active banners, optionally filtered by category
func GetBanners(c *gin.Context) {
	db := database.Database

	fmt.Println("Attempting to fetch banners...")

	// Get category parameter from query string
	categoryID := c.Query("category_id")
	fmt.Printf("Category ID filter: %s\n", categoryID)

	// First, check if the banners table exists
	checkTableQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'banners'
		);
	`
	
	var tableExists bool
	err := db.QueryRow(checkTableQuery).Scan(&tableExists)
	if err != nil {
		fmt.Printf("Error checking if banners table exists: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check banners table",
			"details": err.Error(),
		})
		return
	}

	if !tableExists {
		fmt.Println("Banners table does not exist, creating it...")
		createTableQuery := `
			CREATE TABLE IF NOT EXISTS banners (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				title VARCHAR(255) NOT NULL,
				image_url TEXT NOT NULL,
				link TEXT,
				sort_order INTEGER DEFAULT 0,
				is_active BOOLEAN DEFAULT true,
				category_id UUID REFERENCES categories(id),
				created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
			);
		`
		
		_, err = db.Exec(createTableQuery)
		if err != nil {
			fmt.Printf("Error creating banners table: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create banners table",
				"details": err.Error(),
			})
			return
		}
		fmt.Println("Banners table created successfully")
	}

	// Build query with optional category filter
	var query string
	var args []interface{}
	
	if categoryID != "" {
		query = `
			SELECT id, title, image_url, sort_order, is_active, category_id, created_at, updated_at
			FROM banners 
			WHERE is_active = true 
			AND (category_id = $1 OR category_id IS NULL)
			ORDER BY sort_order ASC, created_at DESC
		`
		args = append(args, categoryID)
	} else {
		query = `
			SELECT id, title, image_url, sort_order, is_active, category_id, created_at, updated_at
			FROM banners 
			WHERE is_active = true 
			ORDER BY sort_order ASC, created_at DESC
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
			"error": "Failed to fetch banners",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var banners []struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		ImageURL   string `json:"image_url"`
		SortOrder  int    `json:"sort_order"`
		IsActive   bool   `json:"is_active"`
		CategoryID *string `json:"category_id"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	}

	for rows.Next() {
		var banner struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			ImageURL   string `json:"image_url"`
			SortOrder  int    `json:"sort_order"`
			IsActive   bool   `json:"is_active"`
			CategoryID *string `json:"category_id"`
			CreatedAt  string `json:"created_at"`
			UpdatedAt  string `json:"updated_at"`
		}

		err := rows.Scan(
			&banner.ID,
			&banner.Title,
			&banner.ImageURL,
			&banner.SortOrder,
			&banner.IsActive,
			&banner.CategoryID,
			&banner.CreatedAt,
			&banner.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("Row scan error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to scan banner data",
				"details": err.Error(),
			})
			return
		}

		banners = append(banners, banner)
	}

	fmt.Printf("Successfully fetched %d banners\n", len(banners))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    banners,
	})
}

// GetAdminBanners returns all banners for admin management
func GetAdminBanners(c *gin.Context) {
	db := database.Database

	query := `
		SELECT id, title, image_url, sort_order, is_active, category_id, created_at, updated_at
		FROM banners 
		ORDER BY sort_order ASC, created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch banners",
		})
		return
	}
	defer rows.Close()

	var banners []struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		ImageURL   string  `json:"image_url"`
		SortOrder  int     `json:"sort_order"`
		IsActive   bool    `json:"is_active"`
		CategoryID *string `json:"category_id"`
		CreatedAt  string  `json:"created_at"`
		UpdatedAt  string  `json:"updated_at"`
		ActionType string  `json:"action_type"`
		ActionData map[string]interface{} `json:"action_data"`
	}

	for rows.Next() {
		var banner struct {
			ID         string  `json:"id"`
			Title      string  `json:"title"`
			ImageURL   string  `json:"image_url"`
			SortOrder  int     `json:"sort_order"`
			IsActive   bool    `json:"is_active"`
			CategoryID *string `json:"category_id"`
			CreatedAt  string  `json:"created_at"`
			UpdatedAt  string  `json:"updated_at"`
			ActionType string  `json:"action_type"`
			ActionData map[string]interface{} `json:"action_data"`
		}

		err := rows.Scan(
			&banner.ID,
			&banner.Title,
			&banner.ImageURL,
			&banner.SortOrder,
			&banner.IsActive,
			&banner.CategoryID,
			&banner.CreatedAt,
			&banner.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to scan banner data",
			})
			return
		}

		// Set default values for action_type and action_data
		banner.ActionType = "search"
		banner.ActionData = map[string]interface{}{
			"type": "search",
		}

		banners = append(banners, banner)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    banners,
	})
}

// CreateBanner creates a new banner
func CreateBanner(c *gin.Context) {
	db := database.Database

	var request struct {
		Title      string `json:"title"`
		ImageURL   string `json:"image_url"`
		SortOrder  int    `json:"sort_order"`
		IsActive   bool   `json:"is_active"`
		CategoryID string `json:"category_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

    // Normalize and validate
    request.ImageURL = strings.TrimSpace(request.ImageURL)
    if strings.HasPrefix(strings.ToLower(request.ImageURL), "http://") {
        request.ImageURL = "https://" + strings.TrimPrefix(request.ImageURL, "http://")
    }

    if request.Title == "" || request.ImageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Title and image URL are required",
		})
		return
	}

	// Insert banner
	query := `
		INSERT INTO banners (title, image_url, sort_order, is_active, category_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	var bannerID string
	var createdAt, updatedAt time.Time

	var categoryID interface{}
	if request.CategoryID != "" {
		categoryID = request.CategoryID
	} else {
		categoryID = nil
	}

    err := db.QueryRow(query,
		request.Title,
		request.ImageURL,
		request.SortOrder,
		request.IsActive,
		categoryID,
	).Scan(&bannerID, &createdAt, &updatedAt)

	if err != nil {
		fmt.Printf("Error creating banner: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create banner",
			"details": err.Error(),
		})
		return
	}

    c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":          bannerID,
			"title":       request.Title,
			"image_url":   request.ImageURL,
			"sort_order":  request.SortOrder,
			"is_active":   request.IsActive,
			"category_id": request.CategoryID,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		},
	})
}

// UpdateBanner updates an existing banner
func UpdateBanner(c *gin.Context) {
	db := database.Database
	bannerID := c.Param("id")

	var request struct {
		Title      string `json:"title"`
		ImageURL   string `json:"image_url"`
		SortOrder  int    `json:"sort_order"`
		IsActive   bool   `json:"is_active"`
		CategoryID string `json:"category_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Validate required fields
	if request.Title == "" || request.ImageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Title and image URL are required",
		})
		return
	}

	// Update banner
	query := `
		UPDATE banners 
		SET title = $1, image_url = $2, sort_order = $3, is_active = $4, category_id = $5, updated_at = now()
		WHERE id = $6
		RETURNING updated_at
	`

	var categoryID interface{}
	if request.CategoryID != "" {
		categoryID = request.CategoryID
	} else {
		categoryID = nil
	}

	var updatedAt time.Time
	err := db.QueryRow(query,
		request.Title,
		request.ImageURL,
		request.SortOrder,
		request.IsActive,
		categoryID,
		bannerID,
	).Scan(&updatedAt)

	if err != nil {
		fmt.Printf("Error updating banner: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update banner",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":          bannerID,
			"title":       request.Title,
			"image_url":   request.ImageURL,
			"sort_order":  request.SortOrder,
			"is_active":   request.IsActive,
			"category_id": request.CategoryID,
			"updated_at":  updatedAt,
		},
	})
}

// UploadBannerImage handles banner image upload to Cloudinary
func UploadBannerImage(c *gin.Context) {
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
	folder := "banners"
    uploadResult, err := services.Cloudinary.UploadImageFromBytes(fileBytes, folder, file.Filename)
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

// GetBannerStats returns banner statistics for admin dashboard
func GetBannerStats(c *gin.Context) {
	db := database.Database

	// Get total banners count
	var totalBanners int
	err := db.QueryRow("SELECT COUNT(*) FROM banners").Scan(&totalBanners)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch total banners count",
		})
		return
	}

	// Get active banners count
	var activeBanners int
	err = db.QueryRow("SELECT COUNT(*) FROM banners WHERE is_active = true").Scan(&activeBanners)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch active banners count",
		})
		return
	}

	// Get inactive banners count
	var inactiveBanners int
	err = db.QueryRow("SELECT COUNT(*) FROM banners WHERE is_active = false").Scan(&inactiveBanners)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch inactive banners count",
		})
		return
	}

	// Get expired banners count (banners with end_date in the past)
	var expiredBanners int
	err = db.QueryRow("SELECT COUNT(*) FROM banners WHERE end_date IS NOT NULL AND end_date < NOW()").Scan(&expiredBanners)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch expired banners count",
		})
		return
	}

	stats := gin.H{
		"total_banners":    totalBanners,
		"active_banners":  activeBanners,
		"inactive_banners": inactiveBanners,
		"expired_banners": expiredBanners,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
