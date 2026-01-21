package services

import (
	"database/sql"
	"fmt"

	"fmbq-server/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// FindMatchingProducts finds bags and shoes that match Melhafa colors
// Returns: matched bag (1), matched shoes (1)
func FindMatchingProducts(melhafColorID uuid.UUID, limitBags int, limitShoes int) ([]map[string]interface{}, []map[string]interface{}, error) {
	if limitBags <= 0 {
		limitBags = 1
	}
	if limitShoes <= 0 {
		limitShoes = 1
	}

	// Get Women category ID (FEMME)
	var womenCategoryID uuid.UUID
	err := database.Database.QueryRow(`
		SELECT id FROM categories 
		WHERE (name ILIKE '%femme%' OR name ILIKE '%women%' OR slug = 'femme')
		AND level = 1 LIMIT 1
	`).Scan(&womenCategoryID)
	if err != nil {
		fmt.Printf("Warning: Failed to find Women category: %v\n", err)
		return []map[string]interface{}{}, []map[string]interface{}{}, fmt.Errorf("failed to find Women category: %w", err)
	}

	// Get Bags and Shoes category IDs under Women
	var bagsCategoryID, shoesCategoryID uuid.UUID
	err = database.Database.QueryRow(`
		SELECT id FROM categories 
		WHERE (name ILIKE '%sac%' OR name ILIKE '%bag%') 
		AND parent_id = $1 AND level = 2 LIMIT 1
	`, womenCategoryID).Scan(&bagsCategoryID)
	if err != nil {
		// Try alternative names
		err2 := database.Database.QueryRow(`
			SELECT id FROM categories 
			WHERE (slug ILIKE '%sac%' OR slug ILIKE '%bag%')
			AND parent_id = $1 LIMIT 1
		`, womenCategoryID).Scan(&bagsCategoryID)
		if err2 != nil {
			fmt.Printf("Warning: Failed to find Bags category: %v\n", err2)
		}
	}

	err = database.Database.QueryRow(`
		SELECT id FROM categories 
		WHERE (name ILIKE '%chaussure%' OR name ILIKE '%shoe%') 
		AND parent_id = $1 AND level = 2 LIMIT 1
	`, womenCategoryID).Scan(&shoesCategoryID)
	if err != nil {
		// Try alternative names
		err2 := database.Database.QueryRow(`
			SELECT id FROM categories 
			WHERE (slug ILIKE '%chaussure%' OR slug ILIKE '%shoe%')
			AND parent_id = $1 LIMIT 1
		`, womenCategoryID).Scan(&shoesCategoryID)
		if err2 != nil {
			fmt.Printf("Warning: Failed to find Shoes category: %v\n", err2)
		}
	}

	// Get canonical colors for this Melhafa color
	canonicalColorQuery := `
		SELECT DISTINCT cc.id, cc.internal_key, cc.name_en, cc.name_fr, cc.color_code
		FROM canonical_colors cc
		JOIN melhaf_color_canonical_matches mccm ON cc.id = mccm.canonical_color_id
		WHERE mccm.melhaf_color_id = $1 AND cc.is_active = TRUE
	`
	canonicalRows, err := database.Database.Query(canonicalColorQuery, melhafColorID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query canonical colors: %w", err)
	}
	defer canonicalRows.Close()

	var canonicalColorIDs []uuid.UUID
	for canonicalRows.Next() {
		var ccID uuid.UUID
		var internalKey, nameEn, nameFr string
		var colorCode sql.NullString
		if err := canonicalRows.Scan(&ccID, &internalKey, &nameEn, &nameFr, &colorCode); err != nil {
			continue
		}
		canonicalColorIDs = append(canonicalColorIDs, ccID)
	}

	if len(canonicalColorIDs) == 0 {
		// No canonical colors matched, return empty results
		fmt.Printf("Info: Melhaf color %s has no canonical colors assigned\n", melhafColorID.String())
		return []map[string]interface{}{}, []map[string]interface{}{}, nil
	}

	fmt.Printf("Info: Found %d canonical colors for Melhaf %s\n", len(canonicalColorIDs), melhafColorID.String())
	fmt.Printf("Info: Looking for bags in category: %s (Women: %s)\n", bagsCategoryID.String(), womenCategoryID.String())

	// Find bags - check both specific bags category and parent FEMME category
	var bags []map[string]interface{}
	if bagsCategoryID != uuid.Nil {
		// Try specific bags category first
		bagsRows, err := database.Database.Query(`
			SELECT DISTINCT
				pm.id,
				pm.title,
				COALESCE(b.name, '') as brand_name,
				pm.description,
				COALESCE(pi.url, '') as image_url,
				COALESCE(pr.list_price, 0) as list_price,
				COALESCE(pr.sale_price, 0) as sale_price,
				pmc.category_id
			FROM product_models pm
			LEFT JOIN brands b ON pm.brand_id = b.id
			JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
			JOIN product_colors pc ON pm.id = pc.product_model_id
			JOIN product_color_canonical_matches pccm ON pc.id = pccm.product_color_id
			LEFT JOIN LATERAL (
				SELECT url FROM product_images 
				WHERE product_images.product_color_id = pc.id 
				ORDER BY product_images.position LIMIT 1
			) pi ON true
			LEFT JOIN LATERAL (
				SELECT p.list_price, p.sale_price 
				FROM skus s
				JOIN prices p ON s.id = p.sku_id
				WHERE s.product_model_id = pm.id 
				ORDER BY p.created_at DESC LIMIT 1
			) pr ON true
			WHERE pccm.canonical_color_id = ANY($1::uuid[])
			AND pm.is_active = TRUE
			AND pmc.category_id = $2
			LIMIT $3
		`, pq.Array(canonicalColorIDs), bagsCategoryID, limitBags)
		if err != nil {
			fmt.Printf("Warning: Failed to query bags: %v\n", err)
		} else {
			defer bagsRows.Close()
			for bagsRows.Next() {
				var id uuid.UUID
				var title, brandName, description sql.NullString
				var imageURL string
				var listPrice, salePrice float64
				var categoryID uuid.UUID

				if err := bagsRows.Scan(&id, &title, &brandName, &description, &imageURL, &listPrice, &salePrice, &categoryID); err != nil {
					continue
				}

				bags = append(bags, map[string]interface{}{
					"id":          id.String(),
					"title":       title.String,
					"brand_name":  brandName.String,
					"description": description.String,
					"image_url":   imageURL,
					"list_price":  listPrice,
					"sale_price":  salePrice,
					"category_id": categoryID.String(),
				})
			}
			fmt.Printf("Info: Found %d matching bags in specific category\n", len(bags))
		}

		// If no bags found in specific category, try parent FEMME category with bag-related products
		if len(bags) == 0 {
			fmt.Printf("Info: No bags found in specific category, trying FEMME parent category and subcategories with bag filter\n")
			bagsRows, err := database.Database.Query(`
				SELECT DISTINCT
					pm.id,
					pm.title,
					COALESCE(b.name, '') as brand_name,
					pm.description,
					COALESCE(pi.url, '') as image_url,
					COALESCE(pr.list_price, 0) as list_price,
					COALESCE(pr.sale_price, 0) as sale_price,
					pmc.category_id
				FROM product_models pm
				LEFT JOIN brands b ON pm.brand_id = b.id
				JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
				JOIN categories c ON pmc.category_id = c.id
				JOIN product_colors pc ON pm.id = pc.product_model_id
				JOIN product_color_canonical_matches pccm ON pc.id = pccm.product_color_id
				LEFT JOIN LATERAL (
					SELECT url FROM product_images 
					WHERE product_images.product_color_id = pc.id 
					ORDER BY product_images.position LIMIT 1
				) pi ON true
				LEFT JOIN LATERAL (
					SELECT p.list_price, p.sale_price 
					FROM skus s
					JOIN prices p ON s.id = p.sku_id
					WHERE s.product_model_id = pm.id 
					ORDER BY p.created_at DESC LIMIT 1
				) pr ON true
				WHERE pccm.canonical_color_id = ANY($1::uuid[])
				AND pm.is_active = TRUE
				AND (c.id = $2 OR c.parent_id = $2)
				AND (pm.title ILIKE '%sac%' OR pm.title ILIKE '%bag%' OR pm.model_code ILIKE '%sac%' OR pm.model_code ILIKE '%bag%' 
				     OR c.name ILIKE '%sac%' OR c.name ILIKE '%bag%' OR c.slug ILIKE '%sac%' OR c.slug ILIKE '%bag%')
				LIMIT $3
			`, pq.Array(canonicalColorIDs), womenCategoryID, limitBags)
			if err == nil {
				defer bagsRows.Close()
				for bagsRows.Next() {
					var id uuid.UUID
					var title, brandName, description sql.NullString
					var imageURL string
					var listPrice, salePrice float64
					var categoryID uuid.UUID

					if err := bagsRows.Scan(&id, &title, &brandName, &description, &imageURL, &listPrice, &salePrice, &categoryID); err != nil {
						continue
					}

					bags = append(bags, map[string]interface{}{
						"id":          id.String(),
						"title":       title.String,
						"brand_name":  brandName.String,
						"description": description.String,
						"image_url":   imageURL,
						"list_price":  listPrice,
						"sale_price":  salePrice,
						"category_id": categoryID.String(),
					})
				}
				fmt.Printf("Info: Found %d matching bags in FEMME category\n", len(bags))
			}
		}
	}

	// Find shoes
	var shoes []map[string]interface{}
	if shoesCategoryID != uuid.Nil {
		fmt.Printf("Info: Looking for shoes in category: %s\n", shoesCategoryID.String())
		shoesRows, err := database.Database.Query(`
			SELECT DISTINCT
				pm.id,
				pm.title,
				COALESCE(b.name, '') as brand_name,
				pm.description,
				COALESCE(pi.url, '') as image_url,
				COALESCE(pr.list_price, 0) as list_price,
				COALESCE(pr.sale_price, 0) as sale_price,
				pmc.category_id
			FROM product_models pm
			LEFT JOIN brands b ON pm.brand_id = b.id
			JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
			JOIN product_colors pc ON pm.id = pc.product_model_id
			JOIN product_color_canonical_matches pccm ON pc.id = pccm.product_color_id
			LEFT JOIN LATERAL (
				SELECT url FROM product_images 
				WHERE product_images.product_color_id = pc.id 
				ORDER BY product_images.position LIMIT 1
			) pi ON true
			LEFT JOIN LATERAL (
				SELECT p.list_price, p.sale_price 
				FROM skus s
				JOIN prices p ON s.id = p.sku_id
				WHERE s.product_model_id = pm.id 
				ORDER BY p.created_at DESC LIMIT 1
			) pr ON true
			WHERE pccm.canonical_color_id = ANY($1::uuid[])
			AND pm.is_active = TRUE
			AND pmc.category_id = $2
			LIMIT $3
		`, pq.Array(canonicalColorIDs), shoesCategoryID, limitShoes)
		if err != nil {
			fmt.Printf("Warning: Failed to query shoes: %v\n", err)
		} else {
			defer shoesRows.Close()
			for shoesRows.Next() {
				var id uuid.UUID
				var title, brandName, description sql.NullString
				var imageURL string
				var listPrice, salePrice float64
				var categoryID uuid.UUID

				if err := shoesRows.Scan(&id, &title, &brandName, &description, &imageURL, &listPrice, &salePrice, &categoryID); err != nil {
					continue
				}

				shoes = append(shoes, map[string]interface{}{
					"id":          id.String(),
					"title":       title.String,
					"brand_name":  brandName.String,
					"description": description.String,
					"image_url":   imageURL,
					"list_price":  listPrice,
					"sale_price":  salePrice,
					"category_id": categoryID.String(),
				})
			}
			fmt.Printf("Info: Found %d matching shoes in specific category\n", len(shoes))
		}

		// If no shoes found in specific category, try parent FEMME category with shoe-related products
		if len(shoes) == 0 {
			fmt.Printf("Info: No shoes found in specific category, trying FEMME parent category and subcategories with shoe filter\n")
			shoesRows, err := database.Database.Query(`
				SELECT DISTINCT
					pm.id,
					pm.title,
					COALESCE(b.name, '') as brand_name,
					pm.description,
					COALESCE(pi.url, '') as image_url,
					COALESCE(pr.list_price, 0) as list_price,
					COALESCE(pr.sale_price, 0) as sale_price,
					pmc.category_id
				FROM product_models pm
				LEFT JOIN brands b ON pm.brand_id = b.id
				JOIN product_model_categories pmc ON pm.id = pmc.product_model_id
				JOIN categories c ON pmc.category_id = c.id
				JOIN product_colors pc ON pm.id = pc.product_model_id
				JOIN product_color_canonical_matches pccm ON pc.id = pccm.product_color_id
				LEFT JOIN LATERAL (
					SELECT url FROM product_images 
					WHERE product_images.product_color_id = pc.id 
					ORDER BY product_images.position LIMIT 1
				) pi ON true
				LEFT JOIN LATERAL (
					SELECT p.list_price, p.sale_price 
					FROM skus s
					JOIN prices p ON s.id = p.sku_id
					WHERE s.product_model_id = pm.id 
					ORDER BY p.created_at DESC LIMIT 1
				) pr ON true
				WHERE pccm.canonical_color_id = ANY($1::uuid[])
				AND pm.is_active = TRUE
				AND (c.id = $2 OR c.parent_id = $2)
				AND (pm.title ILIKE '%chaussure%' OR pm.title ILIKE '%shoe%' OR pm.model_code ILIKE '%chaussure%' OR pm.model_code ILIKE '%shoe%' 
				     OR c.name ILIKE '%chaussure%' OR c.name ILIKE '%shoe%' OR c.slug ILIKE '%chaussure%' OR c.slug ILIKE '%shoe%')
				LIMIT $3
			`, pq.Array(canonicalColorIDs), womenCategoryID, limitShoes)
			if err == nil {
				defer shoesRows.Close()
				for shoesRows.Next() {
					var id uuid.UUID
					var title, brandName, description sql.NullString
					var imageURL string
					var listPrice, salePrice float64
					var categoryID uuid.UUID

					if err := shoesRows.Scan(&id, &title, &brandName, &description, &imageURL, &listPrice, &salePrice, &categoryID); err != nil {
						continue
					}

					shoes = append(shoes, map[string]interface{}{
						"id":          id.String(),
						"title":       title.String,
						"brand_name":  brandName.String,
						"description": description.String,
						"image_url":   imageURL,
						"list_price":  listPrice,
						"sale_price":  salePrice,
						"category_id": categoryID.String(),
					})
				}
				fmt.Printf("Info: Found %d matching shoes in FEMME category\n", len(shoes))
			}
		}
	}

	fmt.Printf("Info: Found %d matching bags, %d matching shoes for Melhaf %s\n", len(bags), len(shoes), melhafColorID.String())

	return bags, shoes, nil
}

// FindMatchingProductsForCollection finds matching products for all colors in a Melhafa collection
// Returns the best matches (one bag, one shoe) based on the most common colors
func FindMatchingProductsForCollection(collectionID uuid.UUID) ([]map[string]interface{}, []map[string]interface{}, error) {
	// Get all colors for this collection
	colorsQuery := `
		SELECT id FROM melhaf_colors 
		WHERE collection_id = $1 AND is_active = TRUE
		ORDER BY sort_order, created_at
	`
	colorsRows, err := database.Database.Query(colorsQuery, collectionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query collection colors: %w", err)
	}
	defer colorsRows.Close()

	var colorIDs []uuid.UUID
	for colorsRows.Next() {
		var colorID uuid.UUID
		if err := colorsRows.Scan(&colorID); err != nil {
			continue
		}
		colorIDs = append(colorIDs, colorID)
	}

	if len(colorIDs) == 0 {
		return []map[string]interface{}{}, []map[string]interface{}{}, nil
	}

	// Try to find matches starting from the first color
	// In production, you might want to aggregate matches from all colors
	if len(colorIDs) > 0 {
		return FindMatchingProducts(colorIDs[0], 1, 1)
	}

	return []map[string]interface{}{}, []map[string]interface{}{}, nil
}
