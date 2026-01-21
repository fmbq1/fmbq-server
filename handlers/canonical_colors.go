package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"fmbq-server/database"
	"fmbq-server/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// AdminGetCanonicalColors handles GET /api/v1/admin/canonical-colors
func AdminGetCanonicalColors(c *gin.Context) {
	rows, err := database.Database.Query(`
		SELECT id, internal_key, name_en, name_fr, color_code, is_active, created_at, updated_at
		FROM canonical_colors
		ORDER BY internal_key ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch canonical colors"})
		return
	}
	defer rows.Close()

	var colors []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var internalKey, nameEn, nameFr string
		var colorCode sql.NullString
		var isActive bool
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &internalKey, &nameEn, &nameFr, &colorCode, &isActive, &createdAt, &updatedAt); err != nil {
			continue
		}

		colors = append(colors, map[string]interface{}{
			"id":           id.String(),
			"internal_key": internalKey,
			"name_en":      nameEn,
			"name_fr":      nameFr,
			"color_code":   colorCode.String,
			"is_active":    isActive,
			"created_at":   createdAt,
			"updated_at":   updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": colors})
}

// AdminCreateCanonicalColor handles POST /api/v1/admin/canonical-colors
func AdminCreateCanonicalColor(c *gin.Context) {
	var req struct {
		InternalKey string  `json:"internal_key" binding:"required"`
		NameEn      string  `json:"name_en" binding:"required"`
		NameFr      string  `json:"name_fr" binding:"required"`
		ColorCode   *string `json:"color_code"`
		IsActive    bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New()
	_, err := database.Database.Exec(`
		INSERT INTO canonical_colors (id, internal_key, name_en, name_fr, color_code, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
	`, id, req.InternalKey, req.NameEn, req.NameFr, req.ColorCode, req.IsActive)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create canonical color"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id": id.String(),
		},
	})
}

// AdminUpdateCanonicalColor handles PUT /api/v1/admin/canonical-colors/:id
func AdminUpdateCanonicalColor(c *gin.Context) {
	id := c.Param("id")
	colorID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid color ID"})
		return
	}

	var req struct {
		InternalKey *string `json:"internal_key"`
		NameEn      *string `json:"name_en"`
		NameFr      *string `json:"name_fr"`
		ColorCode   *string `json:"color_code"`
		IsActive    *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.InternalKey != nil {
		updates = append(updates, fmt.Sprintf("internal_key = $%d", argIndex))
		args = append(args, *req.InternalKey)
		argIndex++
	}
	if req.NameEn != nil {
		updates = append(updates, fmt.Sprintf("name_en = $%d", argIndex))
		args = append(args, *req.NameEn)
		argIndex++
	}
	if req.NameFr != nil {
		updates = append(updates, fmt.Sprintf("name_fr = $%d", argIndex))
		args = append(args, *req.NameFr)
		argIndex++
	}
	if req.ColorCode != nil {
		updates = append(updates, fmt.Sprintf("color_code = $%d", argIndex))
		args = append(args, *req.ColorCode)
		argIndex++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, colorID)

	// Build query properly
	setClause := ""
	for i, update := range updates {
		if i > 0 {
			setClause += ", "
		}
		setClause += update
	}
	
	query := fmt.Sprintf(`
		UPDATE canonical_colors 
		SET %s
		WHERE id = $%d
	`, setClause, argIndex)

	_, err = database.Database.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update canonical color"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminDeleteCanonicalColor handles DELETE /api/v1/admin/canonical-colors/:id
func AdminDeleteCanonicalColor(c *gin.Context) {
	id := c.Param("id")
	colorID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid color ID"})
		return
	}

	_, err = database.Database.Exec("DELETE FROM canonical_colors WHERE id = $1", colorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete canonical color"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminAssignCanonicalColorToMelhaf handles POST /api/v1/admin/melhaf/colors/:id/canonical-colors
func AdminAssignCanonicalColorToMelhaf(c *gin.Context) {
	melhafColorID := c.Param("id")
	colorUUID, err := uuid.Parse(melhafColorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Melhaf color ID"})
		return
	}

	var req struct {
		CanonicalColorIDs []string `json:"canonical_color_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delete existing matches
	_, err = database.Database.Exec(`
		DELETE FROM melhaf_color_canonical_matches WHERE melhaf_color_id = $1
	`, colorUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing matches"})
		return
	}

	// Insert new matches
	for _, canonicalIDStr := range req.CanonicalColorIDs {
		canonicalID, err := uuid.Parse(canonicalIDStr)
		if err != nil {
			continue
		}
		_, err = database.Database.Exec(`
			INSERT INTO melhaf_color_canonical_matches (id, melhaf_color_id, canonical_color_id, created_at)
			VALUES (gen_random_uuid(), $1, $2, now())
			ON CONFLICT (melhaf_color_id, canonical_color_id) DO NOTHING
		`, colorUUID, canonicalID)
		if err != nil {
			fmt.Printf("Warning: Failed to assign canonical color %s: %v\n", canonicalIDStr, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminAssignCanonicalColorToProduct handles POST /api/v1/admin/products/colors/:id/canonical-colors
func AdminAssignCanonicalColorToProduct(c *gin.Context) {
	productColorID := c.Param("id")
	colorUUID, err := uuid.Parse(productColorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product color ID"})
		return
	}

	var req struct {
		CanonicalColorIDs []string `json:"canonical_color_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delete existing matches
	_, err = database.Database.Exec(`
		DELETE FROM product_color_canonical_matches WHERE product_color_id = $1
	`, colorUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear existing matches"})
		return
	}

	// Insert new matches
	for _, canonicalIDStr := range req.CanonicalColorIDs {
		canonicalID, err := uuid.Parse(canonicalIDStr)
		if err != nil {
			continue
		}
		_, err = database.Database.Exec(`
			INSERT INTO product_color_canonical_matches (id, product_color_id, canonical_color_id, created_at)
			VALUES (gen_random_uuid(), $1, $2, now())
			ON CONFLICT (product_color_id, canonical_color_id) DO NOTHING
		`, colorUUID, canonicalID)
		if err != nil {
			fmt.Printf("Warning: Failed to assign canonical color %s: %v\n", canonicalIDStr, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AdminPreviewMelhafMatches handles GET /api/v1/admin/melhaf/colors/:id/matches
// Returns preview of matched bags and shoes for a Melhaf color with canonical colors and debug info
func AdminPreviewMelhafMatches(c *gin.Context) {
	melhafColorID := c.Param("id")
	colorUUID, err := uuid.Parse(melhafColorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Melhaf color ID"})
		return
	}

	// Get Melhaf color info
	var melhafName string
	var colorCode sql.NullString
	database.Database.QueryRow(`SELECT name, color_code FROM melhaf_colors WHERE id = $1`, colorUUID).Scan(&melhafName, &colorCode)

	// Get canonical colors for this Melhaf
	canonicalRows, err := database.Database.Query(`
		SELECT DISTINCT cc.id, cc.internal_key, cc.name_en, cc.name_fr, cc.color_code
		FROM canonical_colors cc
		JOIN melhaf_color_canonical_matches mccm ON cc.id = mccm.canonical_color_id
		WHERE mccm.melhaf_color_id = $1 AND cc.is_active = TRUE
		ORDER BY cc.name_en
	`, colorUUID)

	var canonicalColors []map[string]interface{}
	if err == nil {
		for canonicalRows.Next() {
			var ccID uuid.UUID
			var internalKey, nameEn, nameFr string
			var ccColorCode sql.NullString
			if err := canonicalRows.Scan(&ccID, &internalKey, &nameEn, &nameFr, &ccColorCode); err == nil {
				canonicalColors = append(canonicalColors, map[string]interface{}{
					"id":           ccID.String(),
					"internal_key": internalKey,
					"name_en":      nameEn,
					"name_fr":      nameFr,
					"color_code":   ccColorCode.String,
				})
			}
		}
		canonicalRows.Close()
	}

	bags, shoes, err := services.FindMatchingProducts(colorUUID, 10, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to find matches",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"melhaf_id":        colorUUID.String(),
			"melhaf_name":      melhafName,
			"melhaf_color_code": colorCode.String,
			"canonical_colors": canonicalColors,
			"canonical_colors_count": len(canonicalColors),
			"bags":             bags,
			"bags_count":       len(bags),
			"shoes":            shoes,
			"shoes_count":      len(shoes),
		},
	})
}

// AdminPreviewProductMatches handles GET /api/v1/admin/products/colors/:id/matches
// Returns canonical colors and potential Melhaf matches for a product color
func AdminPreviewProductMatches(c *gin.Context) {
	productColorID := c.Param("id")
	colorUUID, err := uuid.Parse(productColorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product color ID"})
		return
	}

	// Get product color info
	var colorName string
	var colorCode sql.NullString
	var productTitle sql.NullString
	database.Database.QueryRow(`
		SELECT pc.color_name, pc.color_code, pm.title
		FROM product_colors pc
		JOIN product_models pm ON pc.product_model_id = pm.id
		WHERE pc.id = $1
	`, colorUUID).Scan(&colorName, &colorCode, &productTitle)

	// Get canonical colors for this product color
	canonicalRows, err := database.Database.Query(`
		SELECT DISTINCT cc.id, cc.internal_key, cc.name_en, cc.name_fr, cc.color_code
		FROM canonical_colors cc
		JOIN product_color_canonical_matches pccm ON cc.id = pccm.canonical_color_id
		WHERE pccm.product_color_id = $1 AND cc.is_active = TRUE
		ORDER BY cc.name_en
	`, colorUUID)

	var canonicalColors []map[string]interface{}
	if err == nil {
		for canonicalRows.Next() {
			var ccID uuid.UUID
			var internalKey, nameEn, nameFr string
			var ccColorCode sql.NullString
			if err := canonicalRows.Scan(&ccID, &internalKey, &nameEn, &nameFr, &ccColorCode); err == nil {
				canonicalColors = append(canonicalColors, map[string]interface{}{
					"id":           ccID.String(),
					"internal_key": internalKey,
					"name_en":      nameEn,
					"name_fr":      nameFr,
					"color_code":   ccColorCode.String,
				})
			}
		}
		canonicalRows.Close()
	}

	// Find matching Melhafs that share the same canonical colors
	var matchingMelhafs []map[string]interface{}
	if len(canonicalColors) > 0 {
		var canonicalIDs []uuid.UUID
		for _, cc := range canonicalColors {
			if id, err := uuid.Parse(cc["id"].(string)); err == nil {
				canonicalIDs = append(canonicalIDs, id)
			}
		}

		if len(canonicalIDs) > 0 {
			melhafRows, _ := database.Database.Query(`
				SELECT DISTINCT mc.id, mc.name, mc.color_code, mc.price,
				       COALESCE((SELECT url FROM melhaf_color_images WHERE color_id = mc.id ORDER BY position LIMIT 1), '') as image_url
				FROM melhaf_colors mc
				JOIN melhaf_color_canonical_matches mccm ON mc.id = mccm.melhaf_color_id
				WHERE mccm.canonical_color_id = ANY($1::uuid[])
				AND mc.is_active = TRUE
				LIMIT 20
			`, pq.Array(canonicalIDs))

			for melhafRows.Next() {
				var melhafID uuid.UUID
				var melhafName string
				var melhafColorCode sql.NullString
				var price float64
				var imageURL string
				if err := melhafRows.Scan(&melhafID, &melhafName, &melhafColorCode, &price, &imageURL); err == nil {
					matchingMelhafs = append(matchingMelhafs, map[string]interface{}{
						"id":         melhafID.String(),
						"name":       melhafName,
						"color_code": melhafColorCode.String,
						"price":      price,
						"image_url":  imageURL,
					})
				}
			}
			melhafRows.Close()
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"product_color_id":   colorUUID.String(),
			"product_color_name": colorName,
			"product_color_code": colorCode.String,
			"product_title":      productTitle.String,
			"canonical_colors":   canonicalColors,
			"canonical_colors_count": len(canonicalColors),
			"matching_melhafs":   matchingMelhafs,
			"matching_melhafs_count": len(matchingMelhafs),
		},
	})
}

// PublicPreviewProductMatches handles GET /api/v1/public/products/colors/:id/matches
// Public version (no auth required) to check canonical colors and matches for a product color
func PublicPreviewProductMatches(c *gin.Context) {
	// Same implementation as AdminPreviewProductMatches but without auth requirement
	AdminPreviewProductMatches(c)
}

// PublicGetProductColors handles GET /api/v1/public/products/:id/colors
// Returns all colors for a product with their IDs and canonical colors (public, no auth)
func PublicGetProductColors(c *gin.Context) {
	productID := c.Param("id")
	productUUID, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Get product info
	var productTitle sql.NullString
	database.Database.QueryRow(`SELECT title FROM product_models WHERE id = $1`, productUUID).Scan(&productTitle)

	// Get colors with canonical colors from product_colors table
	colorRows, err := database.Database.Query(`
		SELECT 
			pc.id, 
			pc.color_name, 
			COALESCE(pc.color_code, '') as color_code,
			COALESCE(
				(SELECT array_agg(canonical_color_id::text) 
				 FROM product_color_canonical_matches 
				 WHERE product_color_id = pc.id), 
				ARRAY[]::text[]
			) as canonical_color_ids
		FROM product_colors pc 
		WHERE pc.product_model_id = $1 
		ORDER BY pc.created_at
	`, productUUID)

	var colors []map[string]interface{}
	colorMap := make(map[string]map[string]interface{})

	if err == nil {
		defer colorRows.Close()
		for colorRows.Next() {
			var colorID uuid.UUID
			var colorName, colorCode string
			var canonicalColorIDs []string
			if err := colorRows.Scan(&colorID, &colorName, &colorCode, &canonicalColorIDs); err == nil {
				// Get canonical color details
				var canonicalColors []map[string]interface{}
				if len(canonicalColorIDs) > 0 {
					for _, ccIDStr := range canonicalColorIDs {
						ccID, uuidErr := uuid.Parse(ccIDStr)
						if uuidErr == nil {
							var internalKey, nameEn, nameFr string
							var ccColorCode sql.NullString
							if err := database.Database.QueryRow(`
								SELECT id, internal_key, name_en, name_fr, color_code
								FROM canonical_colors
								WHERE id = $1 AND is_active = TRUE
							`, ccID).Scan(&ccID, &internalKey, &nameEn, &nameFr, &ccColorCode); err == nil {
								canonicalColors = append(canonicalColors, map[string]interface{}{
									"id":           ccID.String(),
									"internal_key": internalKey,
									"name_en":      nameEn,
									"name_fr":      nameFr,
									"color_code":   ccColorCode.String,
								})
							}
						}
					}
				}

				colorData := map[string]interface{}{
					"id":                 colorID.String(),
					"color_name":         colorName,
					"color_code":         colorCode,
					"canonical_color_ids": canonicalColorIDs,
					"canonical_colors":   canonicalColors,
					"has_canonical_colors": len(canonicalColorIDs) > 0,
					"source":             "product_colors",
				}
				colors = append(colors, colorData)
				colorMap[colorName] = colorData
			}
		}
	}

	// If no colors found in product_colors, try to derive from SKUs
	if len(colors) == 0 {
		skuRows, err := database.Database.Query(`
			SELECT DISTINCT 
				pc.id,
				pc.color_name,
				COALESCE(pc.color_code, '') as color_code
			FROM skus s
			JOIN product_colors pc ON s.product_color_id = pc.id
			WHERE s.product_model_id = $1
			ORDER BY pc.color_name
		`, productUUID)

		if err == nil {
			defer skuRows.Close()
			for skuRows.Next() {
				var colorID uuid.UUID
				var colorName, colorCode string
				if err := skuRows.Scan(&colorID, &colorName, &colorCode); err == nil {
					// Get canonical colors for this color
					var canonicalColorIDs []string
					ccRows, _ := database.Database.Query(`
						SELECT canonical_color_id::text
						FROM product_color_canonical_matches
						WHERE product_color_id = $1
					`, colorID)
					for ccRows.Next() {
						var ccIDStr string
						if err := ccRows.Scan(&ccIDStr); err == nil {
							canonicalColorIDs = append(canonicalColorIDs, ccIDStr)
						}
					}
					ccRows.Close()

					// Get canonical color details
					var canonicalColors []map[string]interface{}
					if len(canonicalColorIDs) > 0 {
						for _, ccIDStr := range canonicalColorIDs {
							ccID, uuidErr := uuid.Parse(ccIDStr)
							if uuidErr == nil {
								var internalKey, nameEn, nameFr string
								var ccColorCode sql.NullString
								if err := database.Database.QueryRow(`
									SELECT id, internal_key, name_en, name_fr, color_code
									FROM canonical_colors
									WHERE id = $1 AND is_active = TRUE
								`, ccID).Scan(&ccID, &internalKey, &nameEn, &nameFr, &ccColorCode); err == nil {
									canonicalColors = append(canonicalColors, map[string]interface{}{
										"id":           ccID.String(),
										"internal_key": internalKey,
										"name_en":      nameEn,
										"name_fr":      nameFr,
										"color_code":   ccColorCode.String,
									})
								}
							}
						}
					}

					colorData := map[string]interface{}{
						"id":                 colorID.String(),
						"color_name":         colorName,
						"color_code":         colorCode,
						"canonical_color_ids": canonicalColorIDs,
						"canonical_colors":   canonicalColors,
						"has_canonical_colors": len(canonicalColorIDs) > 0,
						"source":             "skus",
					}
					colors = append(colors, colorData)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"product_id":    productID,
			"product_title": productTitle.String,
			"colors":        colors,
			"colors_count":  len(colors),
		},
	})
}
