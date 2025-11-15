package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetAddressBook gets all addresses for a user
func GetAddressBook(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Parse userID string to UUID
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	query := `SELECT id, label, city, quartier, street, building, floor, apartment, 
	          latitude, longitude, is_default, is_active, created_at, updated_at
	          FROM address_book WHERE user_id = $1 AND is_active = true 
	          ORDER BY is_default DESC, created_at DESC`
	
	rows, err := DB.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch addresses"})
		return
	}
	defer rows.Close()

	var addresses []gin.H
	for rows.Next() {
		var addr models.AddressBook
		var street, building, floor, apartment sql.NullString
		var latitude, longitude sql.NullFloat64
		
		err := rows.Scan(
			&addr.ID, &addr.Label, &addr.City, &addr.Quartier,
			&street, &building, &floor, &apartment,
			&latitude, &longitude, &addr.IsDefault, &addr.IsActive,
			&addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			continue
		}

		addressData := gin.H{
			"id":         addr.ID,
			"label":      addr.Label,
			"city":       addr.City,
			"quartier":   addr.Quartier,
			"street":     street.String,
			"building":   building.String,
			"floor":      floor.String,
			"apartment":  apartment.String,
			"latitude":   latitude.Float64,
			"longitude": longitude.Float64,
			"is_default": addr.IsDefault,
			"created_at": addr.CreatedAt,
			"updated_at": addr.UpdatedAt,
		}
		addresses = append(addresses, addressData)
	}

	c.JSON(http.StatusOK, gin.H{"addresses": addresses})
}

// CreateAddress creates a new address (simplified: only city and quartier required)
func CreateAddress(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Parse userID string to UUID
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	var req struct {
		Label     *string  `json:"label,omitempty"`
		City      string   `json:"city" binding:"required"`
		Quartier  string   `json:"quartier" binding:"required"`
		Street    *string  `json:"street,omitempty"`
		Building  *string  `json:"building,omitempty"`
		Floor     *string  `json:"floor,omitempty"`
		Apartment *string  `json:"apartment,omitempty"`
		Latitude  *float64 `json:"latitude,omitempty"`
		Longitude *float64 `json:"longitude,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if this is the first address for this user
	var existingAddressCount int
	countQuery := `SELECT COUNT(*) FROM address_book WHERE user_id = $1 AND is_active = true`
	err = DB.QueryRow(countQuery, userID).Scan(&existingAddressCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing addresses"})
		return
	}

	// First address is automatically set as default
	isDefault := existingAddressCount == 0

	// If this is set as default, unset other defaults
	if isDefault {
		_, err := DB.Exec("UPDATE address_book SET is_default = false WHERE user_id = $1", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update existing defaults"})
			return
		}
	}

	// Auto-generate label if not provided
	label := ""
	if req.Label != nil && *req.Label != "" {
		label = *req.Label
	} else {
		// Generate label: "Address 1", "Address 2", etc.
		label = fmt.Sprintf("Address %d", existingAddressCount+1)
	}

	addressID := uuid.New()
	now := time.Now()
	
	query := `INSERT INTO address_book (id, user_id, label, city, quartier, street, building, 
	          floor, apartment, latitude, longitude, is_default, is_active, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	
	_, err = DB.Exec(query,
		addressID, userID, label, req.City, req.Quartier,
		req.Street, req.Building, req.Floor, req.Apartment,
		req.Latitude, req.Longitude, isDefault, true, now, now,
	)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create address"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Address created successfully",
		"address_id": addressID,
		"is_default": isDefault,
	})
}

// UpdateAddress updates an existing address
func UpdateAddress(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Parse userID string to UUID
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	var req struct {
		City      *string  `json:"city,omitempty"`
		Quartier  *string  `json:"quartier,omitempty"`
		Street    *string  `json:"street,omitempty"`
		Building  *string  `json:"building,omitempty"`
		Floor     *string  `json:"floor,omitempty"`
		Apartment *string  `json:"apartment,omitempty"`
		Latitude  *float64 `json:"latitude,omitempty"`
		Longitude *float64 `json:"longitude,omitempty"`
	}

	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := "UPDATE address_book SET "
	args := []interface{}{}
	argIndex := 1

	if req.City != nil {
		query += "city = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.City)
		argIndex++
	}

	if req.Quartier != nil {
		query += "quartier = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Quartier)
		argIndex++
	}

	if req.Street != nil {
		query += "street = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Street)
		argIndex++
	}

	if req.Building != nil {
		query += "building = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Building)
		argIndex++
	}

	if req.Floor != nil {
		query += "floor = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Floor)
		argIndex++
	}

	if req.Apartment != nil {
		query += "apartment = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Apartment)
		argIndex++
	}

	if req.Latitude != nil {
		query += "latitude = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Latitude)
		argIndex++
	}

	if req.Longitude != nil {
		query += "longitude = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Longitude)
		argIndex++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add updated_at and WHERE clause
	query += "updated_at = $" + strconv.Itoa(argIndex) + " WHERE id = $" + strconv.Itoa(argIndex+1) + " AND user_id = $" + strconv.Itoa(argIndex+2)
	args = append(args, time.Now(), addressID, userID)

	_, err = DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address updated successfully"})
}

// DeleteAddress soft deletes an address
func DeleteAddress(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Parse userID string to UUID
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	_, err = DB.Exec("UPDATE address_book SET is_active = false, updated_at = $1 WHERE id = $2 AND user_id = $3", time.Now(), addressID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address deleted successfully"})
}

// GetCities gets all Mauritanian cities
func GetCities(c *gin.Context) {
	query := `SELECT id, name, name_ar, region FROM cities WHERE is_active = true ORDER BY name`
	
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cities"})
		return
	}
	defer rows.Close()

	var cities []gin.H
	for rows.Next() {
		var city models.City
		err := rows.Scan(&city.ID, &city.Name, &city.NameAr, &city.Region)
		if err != nil {
			continue
		}

		cityData := gin.H{
			"id":      city.ID,
			"name":    city.Name,
			"name_ar": city.NameAr,
			"region":  city.Region,
		}
		cities = append(cities, cityData)
	}

	c.JSON(http.StatusOK, gin.H{"cities": cities})
}

// GetQuartiers gets quartiers for a specific city
func GetQuartiers(c *gin.Context) {
	cityID := c.Param("cityId")
	if cityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "City ID is required"})
		return
	}

	query := `SELECT id, name, name_ar FROM quartiers WHERE city_id = $1 AND is_active = true ORDER BY name`
	
	rows, err := DB.Query(query, cityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quartiers"})
		return
	}
	defer rows.Close()

	var quartiers []gin.H
	for rows.Next() {
		var quartier models.Quartier
		err := rows.Scan(&quartier.ID, &quartier.Name, &quartier.NameAr)
		if err != nil {
			continue
		}

		quartierData := gin.H{
			"id":      quartier.ID,
			"name":    quartier.Name,
			"name_ar": quartier.NameAr,
		}
		quartiers = append(quartiers, quartierData)
	}

	c.JSON(http.StatusOK, gin.H{"quartiers": quartiers})
}
