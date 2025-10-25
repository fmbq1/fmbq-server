package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"fmbq-server/database"

	"github.com/gin-gonic/gin"
)

// Quartier represents a quartier with delivery fee
type Quartier struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	CityID      string  `json:"city_id"`
	CityName    string  `json:"city_name"`
	DeliveryFee float64 `json:"delivery_fee"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}



// UpdateQuartierDeliveryFee handles PUT /api/v1/admin/quartiers/:id/delivery-fee
func UpdateQuartierDeliveryFee(c *gin.Context) {
	quartierID := c.Param("id")
	
	var request struct {
		DeliveryFee float64 `json:"delivery_fee" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		fmt.Printf("❌ Invalid request: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	fmt.Printf("💰 UPDATE DELIVERY FEE - Quartier: %s, Fee: %.2f\n", quartierID, request.DeliveryFee)

	// Validate delivery fee
	if request.DeliveryFee < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Delivery fee cannot be negative"})
		return
	}

	// Update the delivery fee
	query := `
		UPDATE quartiers 
		SET delivery_fee = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, delivery_fee
	`

	var updatedQuartier struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		DeliveryFee float64 `json:"delivery_fee"`
	}

	err := database.Database.QueryRow(query, request.DeliveryFee, quartierID).Scan(
		&updatedQuartier.ID, &updatedQuartier.Name, &updatedQuartier.DeliveryFee,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("❌ Quartier not found: %s\n", quartierID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Quartier not found"})
		} else {
			fmt.Printf("❌ Database error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update delivery fee"})
		}
		return
	}

	fmt.Printf("✅ Delivery fee updated successfully for quartier: %s\n", updatedQuartier.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Delivery fee updated successfully",
		"data":    updatedQuartier,
	})
}

// GetQuartierDeliveryFee handles GET /api/v1/quartiers/:id/delivery-fee
func GetQuartierDeliveryFee(c *gin.Context) {
	quartierID := c.Param("id")
	
	fmt.Printf("💰 GET DELIVERY FEE - Quartier: %s\n", quartierID)

	query := `
		SELECT q.id, q.name, q.city_id, c.name as city_name,
			   COALESCE(q.delivery_fee, 0.00) as delivery_fee
		FROM quartiers q
		LEFT JOIN cities c ON q.city_id = c.id
		WHERE q.id = $1
	`

	var quartier struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		CityID      string  `json:"city_id"`
		CityName    string  `json:"city_name"`
		DeliveryFee float64 `json:"delivery_fee"`
	}

	err := database.Database.QueryRow(query, quartierID).Scan(
		&quartier.ID, &quartier.Name, &quartier.CityID,
		&quartier.CityName, &quartier.DeliveryFee,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("❌ Quartier not found: %s\n", quartierID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Quartier not found"})
		} else {
			fmt.Printf("❌ Database error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch delivery fee"})
		}
		return
	}

	fmt.Printf("✅ Delivery fee: %.2f for quartier: %s\n", quartier.DeliveryFee, quartier.Name)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    quartier,
	})
}

// GetQuartiersByCity handles GET /api/v1/quartiers/city/:cityId
func GetQuartiersByCity(c *gin.Context) {
	cityID := c.Param("cityId")
	
	fmt.Printf("🏘️ GET QUARTIERS BY CITY - City ID: %s\n", cityID)

	query := `
		SELECT q.id, q.name, q.city_id, c.name as city_name,
			   COALESCE(q.delivery_fee, 0.00) as delivery_fee
		FROM quartiers q
		LEFT JOIN cities c ON q.city_id = c.id
		WHERE q.city_id = $1
		ORDER BY q.name
	`

	rows, err := database.Database.Query(query, cityID)
	if err != nil {
		fmt.Printf("❌ Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quartiers"})
		return
	}
	defer rows.Close()

	var quartiers []Quartier
	for rows.Next() {
		var q Quartier
		err := rows.Scan(
			&q.ID, &q.Name, &q.CityID, &q.CityName, &q.DeliveryFee,
		)
		if err != nil {
			fmt.Printf("❌ Scan error: %v\n", err)
			continue
		}
		quartiers = append(quartiers, q)
	}

	fmt.Printf("✅ Found %d quartiers for city %s\n", len(quartiers), cityID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    quartiers,
		"total":   len(quartiers),
	})
}

// GetAdminQuartiers handles GET /api/v1/admin/quartiers - Get all quartiers with delivery fees
func GetAdminQuartiers(c *gin.Context) {
	fmt.Printf("🏘️ GET ALL QUARTIERS - Admin endpoint\n")

	query := `
		SELECT q.id, q.name, q.city_id, c.name as city_name,
			   COALESCE(q.delivery_fee, 0.00) as delivery_fee
		FROM quartiers q
		LEFT JOIN cities c ON q.city_id = c.id
		ORDER BY c.name, q.name
	`

	rows, err := database.Database.Query(query)
	if err != nil {
		fmt.Printf("❌ Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quartiers"})
		return
	}
	defer rows.Close()

	var quartiers []Quartier
	for rows.Next() {
		var q Quartier
		err := rows.Scan(
			&q.ID, &q.Name, &q.CityID, &q.CityName, &q.DeliveryFee,
		)
		if err != nil {
			fmt.Printf("❌ Scan error: %v\n", err)
			continue
		}
		quartiers = append(quartiers, q)
	}

	fmt.Printf("✅ Found %d quartiers total\n", len(quartiers))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    quartiers,
		"total":   len(quartiers),
	})
}

// GetAllQuartiers handles GET /api/v1/quartiers - Get all quartiers with delivery fees (public)
func GetAllQuartiers(c *gin.Context) {
	fmt.Printf("🏘️ GET ALL QUARTIERS - Public endpoint\n")

	query := `
		SELECT q.id, q.name, q.city_id, c.name as city_name,
			   COALESCE(q.delivery_fee, 0.00) as delivery_fee
		FROM quartiers q
		LEFT JOIN cities c ON q.city_id = c.id
		ORDER BY c.name, q.name
	`

	rows, err := database.Database.Query(query)
	if err != nil {
		fmt.Printf("❌ Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quartiers"})
		return
	}
	defer rows.Close()

	var quartiers []Quartier
	for rows.Next() {
		var q Quartier
		err := rows.Scan(
			&q.ID, &q.Name, &q.CityID, &q.CityName, &q.DeliveryFee,
		)
		if err != nil {
			fmt.Printf("❌ Scan error: %v\n", err)
			continue
		}
		quartiers = append(quartiers, q)
	}

	fmt.Printf("✅ Found %d quartiers total\n", len(quartiers))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    quartiers,
		"total":   len(quartiers),
	})
}
