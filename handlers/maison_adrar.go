package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"fmbq-server/database"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminGetMaisonAdrarCategories handles GET /api/v1/admin/maison-adrar/categories
func AdminGetMaisonAdrarCategories(c *gin.Context) {
	rows, err := database.Database.Query(`
		SELECT id, name, name_ar, description, is_active, sort_order, created_at, updated_at
		FROM maison_adrar_categories
		ORDER BY sort_order, name ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	defer rows.Close()

	var categories []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		var nameAr, description sql.NullString
		var isActive bool
		var sortOrder int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &nameAr, &description, &isActive, &sortOrder, &createdAt, &updatedAt); err != nil {
			continue
		}

		var nameArStr *string
		if nameAr.Valid {
			nameArStr = &nameAr.String
		}

		var descriptionStr *string
		if description.Valid {
			descriptionStr = &description.String
		}

		categories = append(categories, map[string]interface{}{
			"id":          id.String(),
			"name":        name,
			"name_ar":     nameArStr,
			"description": descriptionStr,
			"is_active":   isActive,
			"sort_order":  sortOrder,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": categories})
}

// AdminCreateMaisonAdrarCategory handles POST /api/v1/admin/maison-adrar/categories
func AdminCreateMaisonAdrarCategory(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		NameAr      *string `json:"name_ar"`
		Description *string `json:"description"`
		IsActive    bool    `json:"is_active"`
		SortOrder   int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New()
	_, err := database.Database.Exec(`
		INSERT INTO maison_adrar_categories (id, name, name_ar, description, is_active, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
	`, id, req.Name, req.NameAr, req.Description, req.IsActive, req.SortOrder)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id": id.String(),
		},
	})
}

// AdminGetMaisonAdrarCollections handles GET /api/v1/admin/maison-adrar/collections
func AdminGetMaisonAdrarCollections(c *gin.Context) {
	categoryID := c.Query("category_id")

	query := `
		SELECT mc.id, mc.category_id, mc.name, mc.description, mc.background_color, mc.background_url, mc.banner_url,
		       mc.is_active, mc.sort_order, mc.created_at, mc.updated_at,
		       cat.name as category_name
		FROM maison_adrar_collections mc
		LEFT JOIN maison_adrar_categories cat ON mc.category_id = cat.id
	`
	var rows *sql.Rows
	var err error

	if categoryID != "" {
		query += " WHERE mc.category_id = $1"
		rows, err = database.Database.Query(query+" ORDER BY mc.sort_order, mc.name ASC", categoryID)
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
		var id uuid.UUID
		var categoryID sql.NullString
		var name string
		var description, backgroundColor, backgroundURL, bannerURL sql.NullString
		var categoryName sql.NullString
		var isActive bool
		var sortOrder int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &categoryID, &name, &description, &backgroundColor, &backgroundURL, &bannerURL, &isActive, &sortOrder, &createdAt, &updatedAt, &categoryName); err != nil {
			continue
		}

		var backgroundColorStr *string
		if backgroundColor.Valid {
			backgroundColorStr = &backgroundColor.String
		}
		
		var backgroundURLStr *string
		if backgroundURL.Valid {
			backgroundURLStr = &backgroundURL.String
		}
		
		var bannerURLStr *string
		if bannerURL.Valid {
			bannerURLStr = &bannerURL.String
		}

		var descriptionStr *string
		if description.Valid {
			descriptionStr = &description.String
		}

		var categoryIDStr *string
		if categoryID.Valid {
			categoryIDStr = &categoryID.String
		}

		var categoryNameStr *string
		if categoryName.Valid {
			categoryNameStr = &categoryName.String
		}

		collections = append(collections, map[string]interface{}{
			"id":               id.String(),
			"category_id":      categoryIDStr,
			"category_name":    categoryNameStr,
			"name":             name,
			"description":      descriptionStr,
			"background_color": backgroundColorStr,
			"background_url":   backgroundURLStr,
			"banner_url":       bannerURLStr,
			"is_active":        isActive,
			"sort_order":       sortOrder,
			"created_at":       createdAt,
			"updated_at":       updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": collections})
}

// AdminGetMaisonAdrarCollection handles GET /api/v1/admin/maison-adrar/collections/:id
func AdminGetMaisonAdrarCollection(c *gin.Context) {
	id := c.Param("id")
	fmt.Printf("Fetching collection with ID: %s\n", id)

	var collectionID uuid.UUID
	var categoryID sql.NullString
	var name string
	var description, backgroundColor, backgroundURL, bannerURL sql.NullString
	var isActive bool
	var sortOrder int
	var createdAt, updatedAt time.Time
	var categoryName sql.NullString

	err := database.Database.QueryRow(`
		SELECT mc.id, mc.category_id, mc.name, mc.description, mc.background_color, mc.background_url, mc.banner_url,
		       mc.is_active, mc.sort_order, mc.created_at, mc.updated_at,
		       cat.name as category_name
		FROM maison_adrar_collections mc
		LEFT JOIN maison_adrar_categories cat ON mc.category_id = cat.id
		WHERE mc.id = $1
	`, id).Scan(&collectionID, &categoryID, &name, &description, &backgroundColor, &backgroundURL, &bannerURL, &isActive, &sortOrder, &createdAt, &updatedAt, &categoryName)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collection"})
		return
	}

	var backgroundColorStr *string
	if backgroundColor.Valid {
		backgroundColorStr = &backgroundColor.String
	}

	var backgroundURLStr *string
	if backgroundURL.Valid {
		backgroundURLStr = &backgroundURL.String
	}

	var bannerURLStr *string
	if bannerURL.Valid {
		bannerURLStr = &bannerURL.String
	}

	var descriptionStr *string
	if description.Valid {
		descriptionStr = &description.String
	}

	// Get perfumes for this collection
	perfumeRows, err := database.Database.Query(`
		SELECT id, name, name_ar, type, size, description, ingredients, price, discount, is_active, sort_order
		FROM maison_adrar_perfumes
		WHERE collection_id = $1
		ORDER BY sort_order, name ASC
	`, collectionID)

	fmt.Printf("Querying perfumes for collection %s, error: %v\n", collectionID.String(), err)

	var perfumes []map[string]interface{}
	if err == nil {
		for perfumeRows.Next() {
			var perfumeID uuid.UUID
			var perfumeName string
			var nameAr, perfumeType, size, descriptionText, ingredients sql.NullString
			var price float64
			var discount sql.NullFloat64
			var isPerfumeActive bool
			var perfumeSortOrder int

			if err := perfumeRows.Scan(&perfumeID, &perfumeName, &nameAr, &perfumeType, &size, &descriptionText, &ingredients, &price, &discount, &isPerfumeActive, &perfumeSortOrder); err != nil {
				continue
			}

			// Get images for this perfume
			imageRows, imgErr := database.Database.Query(`
				SELECT url 
				FROM maison_adrar_perfume_images 
				WHERE perfume_id = $1 
				ORDER BY position ASC
			`, perfumeID)

			var imageList []string
			if imgErr == nil {
				for imageRows.Next() {
					var imageURL string
					if err := imageRows.Scan(&imageURL); err == nil {
						imageList = append(imageList, imageURL)
					}
				}
				imageRows.Close()
			}

			var nameArStr *string
			if nameAr.Valid {
				nameArStr = &nameAr.String
			}

			var perfumeTypeStr *string
			if perfumeType.Valid {
				perfumeTypeStr = &perfumeType.String
			}

			var sizeStr *string
			if size.Valid {
				sizeStr = &size.String
			}

			var descriptionTextStr *string
			if descriptionText.Valid {
				descriptionTextStr = &descriptionText.String
			}

			var ingredientsStr *string
			if ingredients.Valid {
				ingredientsStr = &ingredients.String
			}

			var discountVal *float64
			if discount.Valid {
				discountVal = &discount.Float64
			}

			perfumes = append(perfumes, map[string]interface{}{
				"id":          perfumeID.String(),
				"name":        perfumeName,
				"name_ar":     nameArStr,
				"type":        perfumeTypeStr,
				"size":        sizeStr,
				"description": descriptionTextStr,
				"ingredients": ingredientsStr,
				"price":       price,
				"discount":    discountVal,
				"is_active":   isPerfumeActive,
				"sort_order":  perfumeSortOrder,
				"images":      imageList,
			})
			fmt.Printf("Found perfume: %s with %d images\n", perfumeName, len(imageList))
		}
		perfumeRows.Close()
	}
	fmt.Printf("Total perfumes found: %d\n", len(perfumes))

	var categoryIDStr *string
	if categoryID.Valid {
		categoryIDStr = &categoryID.String
	}

	var categoryNameStr *string
	if categoryName.Valid {
		categoryNameStr = &categoryName.String
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":               collectionID.String(),
			"category_id":      categoryIDStr,
			"category_name":    categoryNameStr,
			"name":             name,
			"description":      descriptionStr,
			"background_color": backgroundColorStr,
			"background_url":   backgroundURLStr,
			"banner_url":       bannerURLStr,
			"is_active":        isActive,
			"sort_order":       sortOrder,
			"created_at":       createdAt,
			"updated_at":       updatedAt,
			"perfumes":         perfumes,
		},
	})
}

// AdminUpdateMaisonAdrarCollection handles PUT /api/v1/admin/maison-adrar/collections/:id
func AdminUpdateMaisonAdrarCollection(c *gin.Context) {
	id := c.Param("id")
	collectionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	var req struct {
		CategoryID     *string `json:"category_id"`
		Name           string  `json:"name" binding:"required"`
		Description    *string `json:"description"`
		BackgroundColor *string `json:"background_color"`
		IsActive       bool    `json:"is_active"`
		SortOrder      int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var categoryUUID *uuid.UUID
	if req.CategoryID != nil && *req.CategoryID != "" {
		parsed, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category_id"})
			return
		}
		categoryUUID = &parsed
	}

	_, err = database.Database.Exec(`
		UPDATE maison_adrar_collections
		SET category_id = $1, name = $2, description = $3, background_color = $4,
		    is_active = $5, sort_order = $6, updated_at = now()
		WHERE id = $7
	`, categoryUUID, req.Name, req.Description, req.BackgroundColor, req.IsActive, req.SortOrder, collectionID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Collection updated successfully",
	})
}

// AdminCreateMaisonAdrarCollection handles POST /api/v1/admin/maison-adrar/collections
// Creates collection with perfumes and colors
func AdminCreateMaisonAdrarCollection(c *gin.Context) {
	var req struct {
		CategoryID     *string `json:"category_id"`     // Optional - some perfumes have no category
		Name           string  `json:"name" binding:"required"`
		Description    *string `json:"description"`
		BackgroundColor *string `json:"background_color"` // Hex color code
		BackgroundURL  *string `json:"background_url"`
		BannerURL      *string `json:"banner_url"`       // Section banner
		IsActive       bool    `json:"is_active"`
		SortOrder      int     `json:"sort_order"`
		Perfumes       []struct {
			Name        string   `json:"name" binding:"required"`
			NameAr      *string  `json:"name_ar"`
			Type        *string  `json:"type"`        // Eau de Parfum, Eau de Toilette, etc.
			Size        *string  `json:"size"`       // 100ML, 50ML, etc.
			Description *string  `json:"description"`
			Ingredients *string  `json:"ingredients"` // Ingredients list
			Price       float64  `json:"price" binding:"required"`
			Discount    *float64 `json:"discount"`
			SortOrder   int      `json:"sort_order"`
			IsActive    bool     `json:"is_active"`
		} `json:"perfumes" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var categoryUUID *uuid.UUID
	if req.CategoryID != nil && *req.CategoryID != "" {
		parsed, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category_id"})
			return
		}
		categoryUUID = &parsed
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
		INSERT INTO maison_adrar_collections (id, category_id, name, description, background_color, background_url, banner_url, is_active, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())
	`, collectionID, categoryUUID, req.Name, req.Description, req.BackgroundColor, req.BackgroundURL, req.BannerURL, req.IsActive, req.SortOrder)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create collection"})
		return
	}

	// Create perfumes
	var createdPerfumes []map[string]interface{}
	for _, perfumeData := range req.Perfumes {
		perfumeID := uuid.New()

		_, err = tx.Exec(`
			INSERT INTO maison_adrar_perfumes (id, collection_id, name, name_ar, type, size, description, ingredients, price, discount, is_active, sort_order, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now(), now())
		`, perfumeID, collectionID, perfumeData.Name, perfumeData.NameAr, perfumeData.Type, perfumeData.Size, perfumeData.Description, perfumeData.Ingredients, perfumeData.Price, perfumeData.Discount, perfumeData.IsActive, perfumeData.SortOrder)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create perfume: " + perfumeData.Name})
			return
		}

		createdPerfumes = append(createdPerfumes, map[string]interface{}{
			"id":   perfumeID.String(),
			"name": perfumeData.Name,
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
			"id":       collectionID.String(),
			"perfumes": createdPerfumes,
		},
		"message": fmt.Sprintf("Collection created with %d perfumes", len(createdPerfumes)),
	})
}

// AdminCreateMaisonAdrarPerfume handles POST /api/v1/admin/maison-adrar/collections/:id/perfumes
func AdminCreateMaisonAdrarPerfume(c *gin.Context) {
	collectionID := c.Param("id")
	fmt.Printf("Creating perfume for collection: %s\n", collectionID)
	parsedCollectionID, err := uuid.Parse(collectionID)
	if err != nil {
		fmt.Printf("Invalid collection ID: %s\n", collectionID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	var req struct {
		Name        string   `json:"name" binding:"required"`
		NameAr      *string  `json:"name_ar"`
		Type        *string  `json:"type"`
		Size        *string  `json:"size"`
		Description *string  `json:"description"`
		Ingredients *string  `json:"ingredients"`
		Price       float64  `json:"price" binding:"required"`
		Discount    *float64 `json:"discount"`
		SortOrder   int      `json:"sort_order"`
		IsActive    bool     `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("Error binding JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("Creating perfume: %s with price: %f\n", req.Name, req.Price)

	perfumeID := uuid.New()
	_, err = database.Database.Exec(`
		INSERT INTO maison_adrar_perfumes (id, collection_id, name, name_ar, type, size, description, ingredients, price, discount, is_active, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now(), now())
	`, perfumeID, parsedCollectionID, req.Name, req.NameAr, req.Type, req.Size, req.Description, req.Ingredients, req.Price, req.Discount, req.IsActive, req.SortOrder)

	if err != nil {
		fmt.Printf("Error inserting perfume: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create perfume: " + err.Error()})
		return
	}

	fmt.Printf("Perfume created successfully with ID: %s\n", perfumeID.String())

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":   perfumeID.String(),
			"name": req.Name,
		},
		"message": "Perfume created successfully",
	})
}

// AdminUploadMaisonAdrarPerfumeImage handles POST /api/v1/admin/maison-adrar/perfumes/:id/images
func AdminUploadMaisonAdrarPerfumeImage(c *gin.Context) {
	perfumeID := c.Param("id")
	positionStr := c.DefaultPostForm("position", "0")
	alt := c.PostForm("alt")

	position, err := strconv.Atoi(positionStr)
	if err != nil {
		position = 0
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	// Upload to Cloudinary
	imageURL, err := services.UploadMaisonAdrarPerfumeImage(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image: " + err.Error()})
		return
	}

	// Save to database - images are directly linked to perfumes, not colors
	imageID := uuid.New()
	_, err = database.Database.Exec(`
		INSERT INTO maison_adrar_perfume_images (id, perfume_id, url, alt, position, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, imageID, perfumeID, imageURL, alt, position)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":  imageID.String(),
			"url": imageURL,
		},
	})
}

// AdminUploadMaisonAdrarBackground handles POST /api/v1/admin/maison-adrar/collections/:id/background
func AdminUploadMaisonAdrarBackground(c *gin.Context) {
	collectionID := c.Param("id")

	file, err := c.FormFile("background")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No background file provided"})
		return
	}

	// Upload to Cloudinary
	backgroundURL, err := services.UploadMaisonAdrarBackground(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload background: " + err.Error()})
		return
	}

	// Update collection with background URL
	_, err = database.Database.Exec(`
		UPDATE maison_adrar_collections
		SET background_url = $1, updated_at = now()
		WHERE id = $2
	`, backgroundURL, collectionID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"background_url": backgroundURL,
		},
	})
}

// AdminUploadMaisonAdrarBanner handles POST /api/v1/admin/maison-adrar/collections/:id/banner
func AdminUploadMaisonAdrarBanner(c *gin.Context) {
	collectionID := c.Param("id")

	file, err := c.FormFile("banner")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No banner file provided"})
		return
	}

	// Upload to Cloudinary (reuse banner upload service)
	bannerURL, err := services.UploadMaisonAdrarBanner(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload banner: " + err.Error()})
		return
	}

	// Update collection with banner URL
	_, err = database.Database.Exec(`
		UPDATE maison_adrar_collections
		SET banner_url = $1, updated_at = now()
		WHERE id = $2
	`, bannerURL, collectionID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"banner_url": bannerURL,
		},
	})
}

// AdminGetMaisonAdrarBanners handles GET /api/v1/admin/maison-adrar/banners
func AdminGetMaisonAdrarBanners(c *gin.Context) {
	categoryID := c.Query("category_id")

	query := `
		SELECT id, category_id, title, image_url, link_url, is_active, sort_order, created_at, updated_at
		FROM maison_adrar_banners
	`
	var rows *sql.Rows
	var err error

	if categoryID != "" {
		query += " WHERE category_id = $1"
		rows, err = database.Database.Query(query+" ORDER BY sort_order, title ASC", categoryID)
	} else {
		rows, err = database.Database.Query(query + " ORDER BY sort_order, title ASC")
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch banners"})
		return
	}
	defer rows.Close()

	var banners []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var categoryID sql.NullString
		var title string
		var imageURL string
		var linkURL sql.NullString
		var isActive bool
		var sortOrder int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &categoryID, &title, &imageURL, &linkURL, &isActive, &sortOrder, &createdAt, &updatedAt); err != nil {
			continue
		}

		var categoryIDStr *string
		if categoryID.Valid {
			categoryIDStr = &categoryID.String
		}

		var linkURLStr *string
		if linkURL.Valid {
			linkURLStr = &linkURL.String
		}

		banners = append(banners, map[string]interface{}{
			"id":          id.String(),
			"category_id": categoryIDStr,
			"title":       title,
			"image_url":   imageURL,
			"link_url":    linkURLStr,
			"is_active":   isActive,
			"sort_order":  sortOrder,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": banners})
}

// AdminCreateMaisonAdrarBanner handles POST /api/v1/admin/maison-adrar/banners
func AdminCreateMaisonAdrarBanner(c *gin.Context) {
	var req struct {
		CategoryID *string `json:"category_id"`
		Title      string  `json:"title" binding:"required"`
		LinkURL    *string `json:"link_url"`
		IsActive   bool    `json:"is_active"`
		SortOrder  int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get file from multipart form
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	// Upload to Cloudinary
	imageURL, err := services.UploadMaisonAdrarBanner(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload banner image: " + err.Error()})
		return
	}

	var categoryUUID *uuid.UUID
	if req.CategoryID != nil && *req.CategoryID != "" {
		parsed, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category_id"})
			return
		}
		categoryUUID = &parsed
	}

	id := uuid.New()
	_, err = database.Database.Exec(`
		INSERT INTO maison_adrar_banners (id, category_id, title, image_url, link_url, is_active, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
	`, id, categoryUUID, req.Title, imageURL, req.LinkURL, req.IsActive, req.SortOrder)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create banner"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":        id.String(),
			"image_url": imageURL,
		},
	})
}

// GetMaisonAdrarBanners handles GET /api/v1/maison-adrar/banners (public)
func GetMaisonAdrarBanners(c *gin.Context) {
	categoryID := c.Query("category_id")

	query := `
		SELECT b.id, b.category_id, b.title, b.image_url, b.link_url, b.sort_order,
		       cat.id as category_id_uuid, cat.name as category_name
		FROM maison_adrar_banners b
		LEFT JOIN maison_adrar_categories cat ON b.category_id = cat.id
		WHERE b.is_active = true
	`
	var rows *sql.Rows
	var err error

	if categoryID != "" {
		query += " AND (b.category_id = $1 OR b.category_id IS NULL)"
		rows, err = database.Database.Query(query+" ORDER BY b.sort_order, b.title ASC", categoryID)
	} else {
		rows, err = database.Database.Query(query + " ORDER BY b.sort_order, b.title ASC")
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch banners"})
		return
	}
	defer rows.Close()

	var banners []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var categoryID sql.NullString
		var title string
		var imageURL string
		var linkURL sql.NullString
		var sortOrder int
		var categoryIDUUID sql.NullString
		var categoryName sql.NullString

		if err := rows.Scan(&id, &categoryID, &title, &imageURL, &linkURL, &sortOrder, &categoryIDUUID, &categoryName); err != nil {
			continue
		}

		var categoryIDStr *string
		if categoryID.Valid {
			categoryIDStr = &categoryID.String
		}

		var linkURLStr *string
		if linkURL.Valid {
			linkURLStr = &linkURL.String
		}

		var categoryNameStr *string
		if categoryName.Valid {
			categoryNameStr = &categoryName.String
		}

		banners = append(banners, map[string]interface{}{
			"id":           id.String(),
			"category_id":  categoryIDStr,
			"category_name": categoryNameStr,
			"title":        title,
			"image_url":    imageURL,
			"link_url":     linkURLStr,
			"sort_order":   sortOrder,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": banners})
}

// GetMaisonAdrarFeed handles GET /api/v1/maison-adrar/feed (public)
func GetMaisonAdrarFeed(c *gin.Context) {
	categoryID := c.Query("category_id")
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	// Check total collections first
	var totalActive int
	database.Database.QueryRow(`
		SELECT COUNT(*) FROM maison_adrar_collections WHERE is_active = true
	`).Scan(&totalActive)
	fmt.Printf("Total active collections in database: %d\n", totalActive)

	var totalAll int
	database.Database.QueryRow(`
		SELECT COUNT(*) FROM maison_adrar_collections
	`).Scan(&totalAll)
	fmt.Printf("Total collections (all statuses) in database: %d\n", totalAll)

	query := `
		SELECT DISTINCT
			c.id as collection_id,
			c.category_id,
			c.name as collection_name,
			c.description as collection_description,
			c.background_color,
			c.background_url,
			c.banner_url,
			c.sort_order as sort_order,
			cat.name as category_name
		FROM maison_adrar_collections c
		LEFT JOIN maison_adrar_categories cat ON c.category_id = cat.id
		WHERE c.is_active = true
	`
	var rows *sql.Rows
	var queryErr error

	if categoryID != "" {
		query += " AND (c.category_id = $1 OR c.category_id IS NULL)"
		query += " ORDER BY sort_order, collection_name ASC LIMIT $2"
		fmt.Printf("Executing query with categoryID: %s\n", query)
		rows, queryErr = database.Database.Query(query, categoryID, limit)
	} else {
		query += " ORDER BY sort_order, collection_name ASC LIMIT $1"
		fmt.Printf("Executing query without categoryID:\n")
		fmt.Printf("%s\n", query)
		rows, queryErr = database.Database.Query(query, limit)
	}

	if queryErr != nil {
		fmt.Printf("Error fetching Maison Adrar feed: %v\n", queryErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed: " + queryErr.Error()})
		return
	}
	defer rows.Close()

	var collections []map[string]interface{}
	var rowCount int
	
	fmt.Printf("Starting to iterate through rows...\n")
	for rows.Next() {
		rowCount++
		var collectionID uuid.UUID
		var categoryID sql.NullString
		var collectionName string // NOT NULL field
		var collectionDescription, backgroundColor, backgroundURL, bannerURL sql.NullString
		var sortOrder int
		var categoryName sql.NullString

		if err := rows.Scan(&collectionID, &categoryID, &collectionName, &collectionDescription, &backgroundColor, &backgroundURL, &bannerURL, &sortOrder, &categoryName); err != nil {
			fmt.Printf("Error scanning collection row: %v\n", err)
			continue
		}

		// Get perfumes for this collection
		perfumeRows, err := database.Database.Query(`
			SELECT id, name, name_ar, type, size, description, ingredients, price, discount, is_active, sort_order
			FROM maison_adrar_perfumes
			WHERE collection_id = $1 AND is_active = true
			ORDER BY sort_order, name ASC
		`, collectionID)

		if err != nil {
			fmt.Printf("Error fetching perfumes for collection %s: %v\n", collectionID.String(), err)
		}

		var perfumes []map[string]interface{}
		if err == nil {
			for perfumeRows.Next() {
				var perfumeID uuid.UUID
				var name string
				var nameAr, perfumeType, size, description, ingredients sql.NullString
				var price float64
				var discount sql.NullFloat64
				var isActive bool
				var sortOrder int

				if err := perfumeRows.Scan(&perfumeID, &name, &nameAr, &perfumeType, &size, &description, &ingredients, &price, &discount, &isActive, &sortOrder); err != nil {
					continue
				}

				// Get images for this perfume directly
				imageRows, imgErr := database.Database.Query(`
					SELECT url 
					FROM maison_adrar_perfume_images 
					WHERE perfume_id = $1 
					ORDER BY position ASC
				`, perfumeID)

				var imageList []string
				if imgErr == nil {
					for imageRows.Next() {
						var imageURL string
						if err := imageRows.Scan(&imageURL); err == nil {
							imageList = append(imageList, imageURL)
						}
					}
					imageRows.Close()
				}

				var nameArStr *string
				if nameAr.Valid {
					nameArStr = &nameAr.String
				}

				var perfumeTypeStr *string
				if perfumeType.Valid {
					perfumeTypeStr = &perfumeType.String
				}

				var sizeStr *string
				if size.Valid {
					sizeStr = &size.String
				}

				var descriptionStr *string
				if description.Valid {
					descriptionStr = &description.String
				}

				var ingredientsStr *string
				if ingredients.Valid {
					ingredientsStr = &ingredients.String
				}

				var discountVal *float64
				if discount.Valid {
					discountVal = &discount.Float64
				}

				perfumes = append(perfumes, map[string]interface{}{
					"id":          perfumeID.String(),
					"name":        name,
					"name_ar":     nameArStr,
					"type":        perfumeTypeStr,
					"size":        sizeStr,
					"description": descriptionStr,
					"ingredients": ingredientsStr,
					"price":       price,
					"discount":    discountVal,
					"images":      imageList,
				})
			}
			perfumeRows.Close()
		}

		var categoryIDStr *string
		if categoryID.Valid {
			categoryIDStr = &categoryID.String
		}

		var categoryNameStr *string
		if categoryName.Valid {
			categoryNameStr = &categoryName.String
		}

		var collectionDescriptionStr *string
		if collectionDescription.Valid {
			collectionDescriptionStr = &collectionDescription.String
		}

		var backgroundColorStr *string
		if backgroundColor.Valid {
			backgroundColorStr = &backgroundColor.String
		}

		var backgroundURLStr *string
		if backgroundURL.Valid {
			backgroundURLStr = &backgroundURL.String
		}

		var bannerURLStr *string
		if bannerURL.Valid {
			bannerURLStr = &bannerURL.String
		}

		collections = append(collections, map[string]interface{}{
			"id":               collectionID.String(),
			"category_id":      categoryIDStr,
			"category_name":    categoryNameStr,
			"name":             collectionName,
			"description":      collectionDescriptionStr,
			"background_color": backgroundColorStr,
			"background_url":   backgroundURLStr,
			"banner_url":       bannerURLStr,
			"perfumes":         perfumes,
		})
	}
	fmt.Printf("Total rows from rows.Next(): %d, Collections added: %d\n", rowCount, len(collections))

	// Get banners
	bannerRows, err := database.Database.Query(`
		SELECT b.id, b.category_id, b.title, b.image_url, b.link_url, b.sort_order,
		       cat.id as category_id_uuid, cat.name as category_name
		FROM maison_adrar_banners b
		LEFT JOIN maison_adrar_categories cat ON b.category_id = cat.id
		WHERE b.is_active = true
		ORDER BY b.sort_order, b.title ASC
	`)

	var banners []map[string]interface{}
	if err == nil {
		for bannerRows.Next() {
			var id uuid.UUID
			var categoryID sql.NullString
			var title string
			var imageURL string
			var linkURL sql.NullString
			var sortOrder int
			var categoryIDUUID sql.NullString
			var categoryName sql.NullString

			if err := bannerRows.Scan(&id, &categoryID, &title, &imageURL, &linkURL, &sortOrder, &categoryIDUUID, &categoryName); err != nil {
				continue
			}

			var categoryIDStr *string
			if categoryID.Valid {
				categoryIDStr = &categoryID.String
			}

			var linkURLStr *string
			if linkURL.Valid {
				linkURLStr = &linkURL.String
			}

			var categoryNameStr *string
			if categoryName.Valid {
				categoryNameStr = &categoryName.String
			}

			banners = append(banners, map[string]interface{}{
				"id":            id.String(),
				"category_id":   categoryIDStr,
				"category_name": categoryNameStr,
				"title":         title,
				"image_url":     imageURL,
				"link_url":      linkURLStr,
				"sort_order":    sortOrder,
			})
		}
		bannerRows.Close()
	}

	// Get categories
	categoryRows, err := database.Database.Query(`
		SELECT id, name, name_ar, description, sort_order
		FROM maison_adrar_categories
		WHERE is_active = true
		ORDER BY sort_order, name ASC
	`)

	var categories []map[string]interface{}
	if err == nil {
		for categoryRows.Next() {
			var id uuid.UUID
			var name string
			var nameAr, description sql.NullString
			var sortOrder int

			if err := categoryRows.Scan(&id, &name, &nameAr, &description, &sortOrder); err != nil {
				continue
			}

			var nameArStr *string
			if nameAr.Valid {
				nameArStr = &nameAr.String
			}

			var descriptionStr *string
			if description.Valid {
				descriptionStr = &description.String
			}

			categories = append(categories, map[string]interface{}{
				"id":          id.String(),
				"name":        name,
				"name_ar":     nameArStr,
				"description": descriptionStr,
				"sort_order":  sortOrder,
			})
		}
		categoryRows.Close()
	}

	if len(collections) == 0 {
		fmt.Printf("WARNING: No rows returned from query. Active collections count: %d, Total collections: %d\n", totalActive, totalAll)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"collections": collections,
			"categories":  categories,
			"banners":     banners,
		},
	})
}
