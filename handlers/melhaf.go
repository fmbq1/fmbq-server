package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fmbq-server/database"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ==================== ADMIN ENDPOINTS ====================

// AdminGetMelhafTypes handles GET /api/v1/admin/melhaf/types
func AdminGetMelhafTypes(c *gin.Context) {
	rows, err := database.Database.Query(`
		SELECT id, name, name_ar, description, is_active, created_at, updated_at
		FROM melhaf_types
		ORDER BY name ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch melhaf types"})
		return
	}
	defer rows.Close()

	var types []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		var nameAr, description sql.NullString
		var isActive bool
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &nameAr, &description, &isActive, &createdAt, &updatedAt); err != nil {
			continue
		}

		types = append(types, map[string]interface{}{
			"id":          id.String(),
			"name":        name,
			"name_ar":     nameAr.String,
			"description": description.String,
			"is_active":   isActive,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": types})
}

// AdminCreateMelhafType handles POST /api/v1/admin/melhaf/types
func AdminCreateMelhafType(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		NameAr      *string `json:"name_ar"`
		Description *string `json:"description"`
		IsActive    bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New()
	_, err := database.Database.Exec(`
		INSERT INTO melhaf_types (id, name, name_ar, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
	`, id, req.Name, req.NameAr, req.Description, req.IsActive)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create melhaf type"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id": id.String(),
		},
	})
}

// AdminGetMelhafCollections handles GET /api/v1/admin/melhaf/collections
func AdminGetMelhafCollections(c *gin.Context) {
	typeID := c.Query("type_id")

	query := `
		SELECT mc.id, mc.type_id, mc.name, mc.description, mc.is_active, mc.sort_order,
		       mc.created_at, mc.updated_at, mt.name as type_name
		FROM melhaf_collections mc
		JOIN melhaf_types mt ON mc.type_id = mt.id
	`
	var rows *sql.Rows
	var err error

	if typeID != "" {
		query += " WHERE mc.type_id = $1"
		rows, err = database.Database.Query(query+" ORDER BY mc.sort_order, mc.name ASC", typeID)
	} else {
		rows, err = database.Database.Query(query + " ORDER BY mc.sort_order, mc.name ASC")
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collections"})
		return
	}
	defer rows.Close()

	var collections []map[string]interface{}
	for rows.Next() {
		var id, typeID uuid.UUID
		var name, typeName string
		var description sql.NullString
		var isActive bool
		var sortOrder int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &typeID, &name, &description, &isActive, &sortOrder, &createdAt, &updatedAt, &typeName); err != nil {
			continue
		}

		collections = append(collections, map[string]interface{}{
			"id":           id.String(),
			"type_id":      typeID.String(),
			"type_name":    typeName,
			"name":         name,
			"description":  description.String,
			"is_active":    isActive,
			"sort_order":   sortOrder,
			"created_at":   createdAt,
			"updated_at":  updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": collections})
}

// AdminCreateMelhafCollection handles POST /api/v1/admin/melhaf/collections
// Creates collection with colors directly linked to it
func AdminCreateMelhafCollection(c *gin.Context) {
	var req struct {
		TypeID      string `json:"type_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description *string `json:"description"`
		IsActive    bool   `json:"is_active"`
		SortOrder   int    `json:"sort_order"`
		Colors      []struct {
			Name        string   `json:"name" binding:"required"`
			NameAr      *string  `json:"name_ar"`
			ColorCode   *string  `json:"color_code"`
			Price       float64  `json:"price" binding:"required"`
			Discount    *float64 `json:"discount"`
			SortOrder   int      `json:"sort_order"`
			IsActive    bool     `json:"is_active"`
			Inventory   *struct {
				Available    int `json:"available"`
				Reserved     int `json:"reserved"`
				ReorderPoint int `json:"reorder_point"`
			} `json:"inventory"`
		} `json:"colors" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Colors) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collection must have at least one color"})
		return
	}

	typeID, err := uuid.Parse(req.TypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid type_id"})
		return
	}

	// Start transaction
	tx, err := database.Database.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Create collection
	collectionID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO melhaf_collections (id, type_id, name, description, is_active, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
	`, collectionID, typeID, req.Name, req.Description, req.IsActive, req.SortOrder)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create collection"})
		return
	}

	// Create colors for this collection
	var createdColors []map[string]interface{}
	for _, colorData := range req.Colors {
		colorID := uuid.New()
		
		// Auto-generate EAN code (13 digits)
		eanCode := generateMelhafEANCode(req.Name, colorData.Name)

		_, err = tx.Exec(`
			INSERT INTO melhaf_colors (id, collection_id, name, name_ar, color_code, price, discount, ean, is_active, sort_order, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now())
		`, colorID, collectionID, colorData.Name, colorData.NameAr, colorData.ColorCode, colorData.Price, colorData.Discount, eanCode, colorData.IsActive, colorData.SortOrder)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create color: " + colorData.Name})
			return
		}

		// Create inventory record
		available := 0
		reserved := 0
		reorderPoint := 0
		if colorData.Inventory != nil {
			available = colorData.Inventory.Available
			reserved = colorData.Inventory.Reserved
			reorderPoint = colorData.Inventory.ReorderPoint
		}

		_, err = tx.Exec(`
			INSERT INTO melhaf_inventory (id, color_id, available, reserved, reorder_point, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, now(), now())
		`, colorID, available, reserved, reorderPoint)

		if err != nil {
			fmt.Printf("Warning: Failed to create inventory for color %s: %v\n", colorData.Name, err)
		}

		createdColors = append(createdColors, map[string]interface{}{
			"id":   colorID.String(),
			"name": colorData.Name,
			"ean":  eanCode,
		})
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit collection"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":     collectionID.String(),
			"colors": createdColors,
		},
		"message": fmt.Sprintf("Collection created with %d colors", len(createdColors)),
	})
}

// generateMelhafEANCode generates a 13-digit EAN code for melhaf colors
func generateMelhafEANCode(collectionName, colorName string) string {
	// Create a unique string from collection and color info
	baseString := fmt.Sprintf("%s%s%d", collectionName, colorName, time.Now().UnixNano())

	// Convert to numeric representation (only digits)
	var numericString strings.Builder
	for _, char := range baseString {
		if char >= '0' && char <= '9' {
			numericString.WriteRune(char)
		} else if char >= 'A' && char <= 'Z' {
			// Convert letters to numbers (A=1, B=2, etc.)
			numericString.WriteString(fmt.Sprintf("%d", int(char-'A')+1))
		} else if char >= 'a' && char <= 'z' {
			// Convert lowercase letters to numbers
			numericString.WriteString(fmt.Sprintf("%d", int(char-'a')+1))
		}
	}

	// Take last 12 digits (to ensure uniqueness from timestamp)
	base := numericString.String()
	if len(base) > 12 {
		base = base[len(base)-12:]
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

// AdminGetMelhafCollection handles GET /api/v1/admin/melhaf/collections/:id
func AdminGetMelhafCollection(c *gin.Context) {
	id := c.Param("id")
	
	var collectionID uuid.UUID
	var typeID uuid.UUID
	var name string
	var description sql.NullString
	var isActive bool
	var sortOrder int
	var createdAt, updatedAt time.Time
	var typeName string

	err := database.Database.QueryRow(`
		SELECT mc.id, mc.type_id, mc.name, mc.description, mc.is_active, mc.sort_order,
		       mc.created_at, mc.updated_at, mt.name as type_name
		FROM melhaf_collections mc
		JOIN melhaf_types mt ON mc.type_id = mt.id
		WHERE mc.id = $1
	`, id).Scan(&collectionID, &typeID, &name, &description, &isActive, &sortOrder, &createdAt, &updatedAt, &typeName)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":           collectionID.String(),
			"type_id":      typeID.String(),
			"type_name":    typeName,
			"name":         name,
			"description":  description.String,
			"is_active":    isActive,
			"sort_order":   sortOrder,
			"created_at":   createdAt,
			"updated_at":   updatedAt,
		},
	})
}

// AdminUpdateMelhafCollection handles PUT /api/v1/admin/melhaf/collections/:id
func AdminUpdateMelhafCollection(c *gin.Context) {
	id := c.Param("id")
	
	var req struct {
		TypeID      string  `json:"type_id" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		Description *string `json:"description"`
		IsActive    bool    `json:"is_active"`
		SortOrder   int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	typeID, err := uuid.Parse(req.TypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid type_id"})
		return
	}

	collectionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection id"})
		return
	}

	// Check if collection exists
	var exists bool
	err = database.Database.QueryRow(`SELECT EXISTS(SELECT 1 FROM melhaf_collections WHERE id = $1)`, collectionID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}

	// Update collection
	_, err = database.Database.Exec(`
		UPDATE melhaf_collections
		SET type_id = $1, name = $2, description = $3, is_active = $4, sort_order = $5, updated_at = now()
		WHERE id = $6
	`, typeID, req.Name, req.Description, req.IsActive, req.SortOrder, collectionID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id": collectionID.String(),
		},
		"message": "Collection updated successfully",
	})
}

// AdminGetMelhafColors handles GET /api/v1/admin/melhaf/colors
func AdminGetMelhafColors(c *gin.Context) {
	collectionID := c.Query("collection_id")

	query := `
		SELECT mc.id, mc.collection_id, mc.name, mc.name_ar, mc.color_code, 
		       mc.price, mc.discount, mc.ean, mc.is_active, mc.sort_order,
		       mc.created_at, mc.updated_at
		FROM melhaf_colors mc
	`
	var rows *sql.Rows
	var err error

	if collectionID != "" {
		query += " WHERE mc.collection_id = $1"
		rows, err = database.Database.Query(query+" ORDER BY mc.sort_order, mc.name ASC", collectionID)
	} else {
		rows, err = database.Database.Query(query + " ORDER BY mc.sort_order, mc.name ASC")
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch colors"})
		return
	}
	defer rows.Close()

	var colors []map[string]interface{}
	for rows.Next() {
		var id, collectionID uuid.UUID
		var name string
		var nameAr, colorCode, ean sql.NullString
		var price float64
		var discount sql.NullFloat64
		var isActive bool
		var sortOrder int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &collectionID, &name, &nameAr, &colorCode, &price, &discount, &ean, &isActive, &sortOrder, &createdAt, &updatedAt); err != nil {
			continue
		}

		colors = append(colors, map[string]interface{}{
			"id":            id.String(),
			"collection_id": collectionID.String(),
			"name":          name,
			"name_ar":       nameAr.String,
			"color_code":    colorCode.String,
			"price":         price,
			"discount":      discount.Float64,
			"ean":           ean.String,
			"is_active":     isActive,
			"sort_order":    sortOrder,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": colors})
}

// AdminCreateMelhafColor handles POST /api/v1/admin/melhaf/colors
// This is for adding individual colors to existing collections
func AdminCreateMelhafColor(c *gin.Context) {
	var req struct {
		CollectionID string   `json:"collection_id" binding:"required"`
		Name          string   `json:"name" binding:"required"`
		NameAr        *string  `json:"name_ar"`
		ColorCode     *string  `json:"color_code"`
		Price         float64  `json:"price" binding:"required"`
		Discount      *float64 `json:"discount"`
		IsActive      bool     `json:"is_active"`
		SortOrder     int      `json:"sort_order"`
		Inventory     *struct {
			Available    int `json:"available"`
			Reserved     int `json:"reserved"`
			ReorderPoint int `json:"reorder_point"`
		} `json:"inventory"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collectionID, err := uuid.Parse(req.CollectionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection_id"})
		return
	}

	// Get collection name for EAN generation
	var collectionName string
	err = database.Database.QueryRow(`SELECT name FROM melhaf_collections WHERE id = $1`, collectionID).Scan(&collectionName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collection not found"})
		return
	}

	// Auto-generate EAN code (13 digits)
	eanCode := generateMelhafEANCode(collectionName, req.Name)

	// Start transaction
	tx, err := database.Database.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	colorID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO melhaf_colors (id, collection_id, name, name_ar, color_code, price, discount, ean, is_active, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now())
	`, colorID, collectionID, req.Name, req.NameAr, req.ColorCode, req.Price, req.Discount, eanCode, req.IsActive, req.SortOrder)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create color"})
		return
	}

	// Create inventory entry
	available := 0
	reserved := 0
	reorderPoint := 0
	if req.Inventory != nil {
		available = req.Inventory.Available
		reserved = req.Inventory.Reserved
		reorderPoint = req.Inventory.ReorderPoint
	}

	_, err = tx.Exec(`
		INSERT INTO melhaf_inventory (id, color_id, available, reserved, reorder_point, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, now(), now())
	`, colorID, available, reserved, reorderPoint)

	if err != nil {
		fmt.Printf("Warning: Failed to create inventory entry: %v\n", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit color"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":  colorID.String(),
			"ean": eanCode,
		},
		"message": "Color created with auto-generated EAN code",
	})
}

// AdminUploadMelhafImage handles POST /api/v1/admin/melhaf/colors/:id/images
// Uploads image to Cloudinary and stores URL in database
func AdminUploadMelhafImage(c *gin.Context) {
	colorID := c.Param("id")
	
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image file is required"})
		return
	}

	// Upload to Cloudinary using the service
	imageURL, err := services.UploadImageToCloudinary(file)
	if err != nil {
		fmt.Printf("❌ Cloudinary upload error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to Cloudinary: " + err.Error()})
		return
	}

	fmt.Printf("✅ Image uploaded to Cloudinary: %s\n", imageURL)

	position := c.DefaultPostForm("position", "0")
	pos, _ := strconv.Atoi(position)
	alt := c.PostForm("alt")

	id := uuid.New()
	_, err = database.Database.Exec(`
		INSERT INTO melhaf_color_images (id, color_id, url, alt, position, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, id, colorID, imageURL, alt, pos)

	if err != nil {
		fmt.Printf("❌ Database insert error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image URL to database"})
		return
	}

	fmt.Printf("✅ Image URL saved to database for color %s\n", colorID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":  id.String(),
			"url": imageURL,
		},
		"message": "Image uploaded to Cloudinary and saved successfully",
	})
}

// AdminUploadMelhafVideo handles POST /api/v1/admin/melhaf/videos
func AdminUploadMelhafVideo(c *gin.Context) {
	collectionID := c.PostForm("collection_id")
	title := c.PostForm("title")
	description := c.PostForm("description")

	if collectionID == "" || title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection_id and title are required"})
		return
	}

	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video file is required"})
		return
	}

	// Upload video to Cloudinary
	videoURL, err := services.UploadVideoToCloudinary(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload video"})
		return
	}

	// Force HTTPS
	videoURL = strings.Replace(videoURL, "http://", "https://", 1)

	sortOrder := c.DefaultPostForm("sort_order", "0")
	sort, _ := strconv.Atoi(sortOrder)
	isActive := c.DefaultPostForm("is_active", "true") == "true"

	id := uuid.New()
	_, err = database.Database.Exec(`
		INSERT INTO melhaf_videos (id, collection_id, title, description, video_url, is_active, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
	`, id, collectionID, title, description, videoURL, isActive, sort)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":        id.String(),
			"video_url": videoURL,
		},
	})
}

// AdminUpdateMelhafInventory handles PUT /api/v1/admin/melhaf/colors/:id/inventory
func AdminUpdateMelhafInventory(c *gin.Context) {
	colorID := c.Param("id")

	var req struct {
		Available    int `json:"available"`
		Reserved     int `json:"reserved"`
		ReorderPoint int `json:"reorder_point"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Database.Exec(`
		UPDATE melhaf_inventory 
		SET available = $1, reserved = $2, reorder_point = $3, updated_at = now()
		WHERE color_id = $4
	`, req.Available, req.Reserved, req.ReorderPoint, colorID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update inventory"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ==================== PUBLIC ENDPOINTS ====================

// GetMelhafVideos handles GET /api/v1/melhaf/videos
func GetMelhafVideos(c *gin.Context) {
	collectionID := c.Query("collection_id")
	limit := c.DefaultQuery("limit", "20")
	limitInt, _ := strconv.Atoi(limit)
	if limitInt > 50 {
		limitInt = 50
	}

	query := `
		SELECT mv.id, mv.collection_id, mv.title, mv.description, mv.video_url, 
		       mv.thumbnail_url, mv.duration, mv.is_active, mv.sort_order,
		       mv.created_at, mv.updated_at, mc.name as collection_name
		FROM melhaf_videos mv
		JOIN melhaf_collections mc ON mv.collection_id = mc.id
		WHERE mv.is_active = true
	`
	var rows *sql.Rows
	var err error

	if collectionID != "" {
		query += " AND mv.collection_id = $1"
		rows, err = database.Database.Query(query+" ORDER BY mv.sort_order, mv.created_at DESC LIMIT $2", collectionID, limitInt)
	} else {
		rows, err = database.Database.Query(query+" ORDER BY mv.sort_order, mv.created_at DESC LIMIT $1", limitInt)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch videos"})
		return
	}
	defer rows.Close()

	var videos []map[string]interface{}
	for rows.Next() {
		var id, collectionID uuid.UUID
		var title, videoURL, collectionName string
		var description, thumbnailURL sql.NullString
		var duration sql.NullInt32
		var isActive bool
		var sortOrder int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &collectionID, &title, &description, &videoURL, &thumbnailURL, &duration, &isActive, &sortOrder, &createdAt, &updatedAt, &collectionName); err != nil {
			continue
		}

		// Get collection colors
		colorRows, _ := database.Database.Query(`
			SELECT mc.id, mc.name, mc.name_ar, mc.color_code, mc.price, mc.discount,
			       COALESCE((SELECT url FROM melhaf_color_images WHERE color_id = mc.id ORDER BY position LIMIT 1), '') as image_url
			FROM melhaf_colors mc
			WHERE mc.collection_id = $1 AND mc.is_active = true
			ORDER BY mc.sort_order
		`, collectionID)
		
		var colors []map[string]interface{}
		for colorRows.Next() {
			var colorID uuid.UUID
			var colorName string
			var nameAr, colorCode sql.NullString
			var price float64
			var discount sql.NullFloat64
			var imageURL sql.NullString

			if err := colorRows.Scan(&colorID, &colorName, &nameAr, &colorCode, &price, &discount, &imageURL); err == nil {
				colors = append(colors, map[string]interface{}{
					"id":         colorID.String(),
					"name":       colorName,
					"name_ar":    nameAr.String,
					"color_code": colorCode.String,
					"price":      price,
					"discount":   discount.Float64,
					"image_url":  imageURL.String,
				})
			}
		}
		colorRows.Close()

		// Get like count
		var likeCount int
		database.Database.QueryRow(
			"SELECT COUNT(*) FROM melhaf_video_likes WHERE video_id = $1",
			id,
		).Scan(&likeCount)

		// Get reaction counts
		reactionCounts := make(map[string]int)
		reactionRows, _ := database.Database.Query(
			"SELECT reaction, COUNT(*) FROM melhaf_video_reactions WHERE video_id = $1 GROUP BY reaction",
			id,
		)
		defer reactionRows.Close()
		for reactionRows.Next() {
			var reaction string
			var count int
			if err := reactionRows.Scan(&reaction, &count); err == nil {
				reactionCounts[reaction] = count
			}
		}

		// Get user's interactions if authenticated
		var userLiked bool
		var userReactions []string
		userIDStr, exists := c.Get("user_id")
		if exists {
			userID, _ := uuid.Parse(userIDStr.(string))
			database.Database.QueryRow(
				"SELECT EXISTS(SELECT 1 FROM melhaf_video_likes WHERE video_id = $1 AND user_id = $2)",
				id, userID,
			).Scan(&userLiked)

			userReactionRows, _ := database.Database.Query(
				"SELECT reaction FROM melhaf_video_reactions WHERE video_id = $1 AND user_id = $2",
				id, userID,
			)
			defer userReactionRows.Close()
			for userReactionRows.Next() {
				var reaction string
				if err := userReactionRows.Scan(&reaction); err == nil {
					userReactions = append(userReactions, reaction)
				}
			}
		}

		videos = append(videos, map[string]interface{}{
			"id":              id.String(),
			"collection_id":   collectionID.String(),
			"collection_name": collectionName,
			"title":           title,
			"description":     description.String,
			"video_url":       videoURL,
			"thumbnail_url":   thumbnailURL.String,
			"duration":        duration.Int32,
			"colors":          colors,
			"created_at":      createdAt,
			"like_count":      likeCount,
			"user_liked":      userLiked,
			"reaction_counts": reactionCounts,
			"user_reactions":  userReactions,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": videos})
}

// ==================== VIDEO LIKES & REACTIONS ====================

// LikeMelhafVideo handles POST /api/v1/melhaf/videos/:id/like
func LikeMelhafVideo(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	videoIDStr := c.Param("id")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	// Check if like already exists
	var existingLikeID uuid.UUID
	err = database.Database.QueryRow(
		"SELECT id FROM melhaf_video_likes WHERE video_id = $1 AND user_id = $2",
		videoID, userID,
	).Scan(&existingLikeID)

	if err == nil {
		// Unlike: delete the like
		_, err = database.Database.Exec(
			"DELETE FROM melhaf_video_likes WHERE id = $1",
			existingLikeID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlike video"})
			return
		}

		// Get updated like count
		var likeCount int
		database.Database.QueryRow(
			"SELECT COUNT(*) FROM melhaf_video_likes WHERE video_id = $1",
			videoID,
		).Scan(&likeCount)

		c.JSON(http.StatusOK, gin.H{
			"success":   true,
			"liked":    false,
			"like_count": likeCount,
		})
		return
	}

	// Like: create new like
	likeID := uuid.New()
	_, err = database.Database.Exec(
		"INSERT INTO melhaf_video_likes (id, video_id, user_id, created_at) VALUES ($1, $2, $3, now())",
		likeID, videoID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like video"})
		return
	}

	// Get updated like count
	var likeCount int
	database.Database.QueryRow(
		"SELECT COUNT(*) FROM melhaf_video_likes WHERE video_id = $1",
		videoID,
	).Scan(&likeCount)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"liked":      true,
		"like_count": likeCount,
	})
}

// ReactToMelhafVideo handles POST /api/v1/melhaf/videos/:id/react
func ReactToMelhafVideo(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	videoIDStr := c.Param("id")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	var req struct {
		Reaction string `json:"reaction" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reaction emoji required"})
		return
	}

	// Validate reaction is a single emoji (basic check)
	if len([]rune(req.Reaction)) > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reaction format"})
		return
	}

	// Check if reaction already exists
	var existingReactionID uuid.UUID
	err = database.Database.QueryRow(
		"SELECT id FROM melhaf_video_reactions WHERE video_id = $1 AND user_id = $2 AND reaction = $3",
		videoID, userID, req.Reaction,
	).Scan(&existingReactionID)

	if err == nil {
		// Remove reaction: delete it
		_, err = database.Database.Exec(
			"DELETE FROM melhaf_video_reactions WHERE id = $1",
			existingReactionID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove reaction"})
			return
		}

		// Get updated reaction counts
		reactionCounts := make(map[string]int)
		rows, _ := database.Database.Query(
			"SELECT reaction, COUNT(*) FROM melhaf_video_reactions WHERE video_id = $1 GROUP BY reaction",
			videoID,
		)
		defer rows.Close()
		for rows.Next() {
			var reaction string
			var count int
			if err := rows.Scan(&reaction, &count); err == nil {
				reactionCounts[reaction] = count
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success":         true,
			"reacted":        false,
			"reaction":       req.Reaction,
			"reaction_counts": reactionCounts,
		})
		return
	}

	// Add reaction: create new reaction
	reactionID := uuid.New()
	_, err = database.Database.Exec(
		"INSERT INTO melhaf_video_reactions (id, video_id, user_id, reaction, created_at) VALUES ($1, $2, $3, $4, now())",
		reactionID, videoID, userID, req.Reaction,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reaction"})
		return
	}

	// Get updated reaction counts
	reactionCounts := make(map[string]int)
	rows, _ := database.Database.Query(
		"SELECT reaction, COUNT(*) FROM melhaf_video_reactions WHERE video_id = $1 GROUP BY reaction",
		videoID,
	)
	defer rows.Close()
	for rows.Next() {
		var reaction string
		var count int
		if err := rows.Scan(&reaction, &count); err == nil {
			reactionCounts[reaction] = count
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"reacted":         true,
		"reaction":        req.Reaction,
		"reaction_counts": reactionCounts,
	})
}

// GetVideoInteractions handles GET /api/v1/melhaf/videos/:id/interactions
func GetVideoInteractions(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	// Get like count
	var likeCount int
	database.Database.QueryRow(
		"SELECT COUNT(*) FROM melhaf_video_likes WHERE video_id = $1",
		videoID,
	).Scan(&likeCount)

	// Get reaction counts
	reactionCounts := make(map[string]int)
	rows, _ := database.Database.Query(
		"SELECT reaction, COUNT(*) FROM melhaf_video_reactions WHERE video_id = $1 GROUP BY reaction",
		videoID,
	)
	defer rows.Close()
	for rows.Next() {
		var reaction string
		var count int
		if err := rows.Scan(&reaction, &count); err == nil {
			reactionCounts[reaction] = count
		}
	}

	// Get user's interactions if authenticated
	var userLiked bool
	var userReactions []string
	userIDStr, exists := c.Get("user_id")
	if exists {
		userID, _ := uuid.Parse(userIDStr.(string))
		database.Database.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM melhaf_video_likes WHERE video_id = $1 AND user_id = $2)",
			videoID, userID,
		).Scan(&userLiked)

		reactionRows, _ := database.Database.Query(
			"SELECT reaction FROM melhaf_video_reactions WHERE video_id = $1 AND user_id = $2",
			videoID, userID,
		)
		defer reactionRows.Close()
		for reactionRows.Next() {
			var reaction string
			if err := reactionRows.Scan(&reaction); err == nil {
				userReactions = append(userReactions, reaction)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"like_count":      likeCount,
		"user_liked":      userLiked,
		"reaction_counts": reactionCounts,
		"user_reactions":  userReactions,
	})
}

// GetMelhafColorDetails handles GET /api/v1/melhaf/colors/:id
func GetMelhafColorDetails(c *gin.Context) {
	colorID := c.Param("id")

	var color struct {
		ID           uuid.UUID
		CollectionID uuid.UUID
		Name         string
		NameAr       sql.NullString
		ColorCode    sql.NullString
		Price        float64
		Discount     sql.NullFloat64
		EAN          sql.NullString
		CollectionName string
	}

	err := database.Database.QueryRow(`
		SELECT mc.id, mc.collection_id, mc.name, mc.name_ar, mc.color_code, 
		       mc.price, mc.discount, mc.ean, mc.name as collection_name
		FROM melhaf_colors mc
		JOIN melhaf_collections mcol ON mc.collection_id = mcol.id
		WHERE mc.id = $1 AND mc.is_active = true
	`, colorID).Scan(&color.ID, &color.CollectionID, &color.Name, &color.NameAr, &color.ColorCode, &color.Price, &color.Discount, &color.EAN, &color.CollectionName)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Color not found"})
		return
	}

	// Get images
	imageRows, _ := database.Database.Query(`
		SELECT id, url, alt, position
		FROM melhaf_color_images
		WHERE color_id = $1
		ORDER BY position
	`, colorID)

	var images []map[string]interface{}
	for imageRows.Next() {
		var imgID uuid.UUID
		var url string
		var alt sql.NullString
		var position int

		if err := imageRows.Scan(&imgID, &url, &alt, &position); err == nil {
			images = append(images, map[string]interface{}{
				"id":       imgID.String(),
				"url":      url,
				"alt":      alt.String,
				"position": position,
			})
		}
	}
	imageRows.Close()

	// Get inventory
	var available, reserved int
	database.Database.QueryRow(`
		SELECT available, reserved
		FROM melhaf_inventory
		WHERE color_id = $1
	`, colorID).Scan(&available, &reserved)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":             color.ID.String(),
			"collection_id":  color.CollectionID.String(),
			"collection_name": color.CollectionName,
			"name":           color.Name,
			"name_ar":        color.NameAr.String,
			"color_code":     color.ColorCode.String,
			"price":          color.Price,
			"discount":       color.Discount.Float64,
			"ean":            color.EAN.String,
			"images":         images,
			"available":     available,
			"reserved":       reserved,
		},
	})
}

