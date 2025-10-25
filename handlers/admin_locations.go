package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminGetCities gets all cities for admin management
func AdminGetCities(c *gin.Context) {
	query := `SELECT id, name, name_ar, region, is_active, created_at FROM cities ORDER BY name`
	
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cities"})
		return
	}
	defer rows.Close()

	var cities []gin.H
	for rows.Next() {
		var city models.City
		var nameAr sql.NullString
		var createdAt time.Time
		
		err := rows.Scan(&city.ID, &city.Name, &nameAr, &city.Region, &city.IsActive, &createdAt)
		if err != nil {
			continue
		}

		cityData := gin.H{
			"id":         city.ID,
			"name":       city.Name,
			"name_ar":    nameAr.String,
			"region":     city.Region,
			"is_active":  city.IsActive,
			"created_at": createdAt,
		}
		cities = append(cities, cityData)
	}

	c.JSON(http.StatusOK, gin.H{"cities": cities})
}

// AdminCreateCity creates a new city
func AdminCreateCity(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		NameAr  string `json:"name_ar,omitempty"`
		Region  string `json:"region" binding:"required"`
		IsActive bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cityID := uuid.New()
	now := time.Now()
	
	query := `INSERT INTO cities (id, name, name_ar, region, is_active, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err := DB.Exec(query, cityID, req.Name, req.NameAr, req.Region, req.IsActive, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create city"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "City created successfully",
		"city_id": cityID,
	})
}

// AdminUpdateCity updates an existing city
func AdminUpdateCity(c *gin.Context) {
	cityID := c.Param("id")
	if cityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "City ID is required"})
		return
	}

	var req struct {
		Name     *string `json:"name,omitempty"`
		NameAr   *string `json:"name_ar,omitempty"`
		Region   *string `json:"region,omitempty"`
		IsActive *bool   `json:"is_active,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := "UPDATE cities SET "
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		query += "name = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Name)
		argIndex++
	}

	if req.NameAr != nil {
		query += "name_ar = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.NameAr)
		argIndex++
	}

	if req.Region != nil {
		query += "region = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Region)
		argIndex++
	}

	if req.IsActive != nil {
		query += "is_active = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add updated_at and WHERE clause
	query += "updated_at = $" + strconv.Itoa(argIndex) + " WHERE id = $" + strconv.Itoa(argIndex+1)
	args = append(args, time.Now(), cityID)

	_, err := DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update city"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "City updated successfully"})
}

// AdminDeleteCity deletes a city (soft delete)
func AdminDeleteCity(c *gin.Context) {
	cityID := c.Param("id")
	if cityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "City ID is required"})
		return
	}

	_, err := DB.Exec("UPDATE cities SET is_active = false, updated_at = $1 WHERE id = $2", time.Now(), cityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete city"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "City deleted successfully"})
}

// AdminGetQuartiers gets all quartiers for a city
func AdminGetQuartiers(c *gin.Context) {
	cityID := c.Param("cityId")
	if cityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "City ID is required"})
		return
	}

	query := `SELECT id, name, name_ar, is_active, created_at FROM quartiers 
	          WHERE city_id = $1 ORDER BY name`
	
	rows, err := DB.Query(query, cityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quartiers"})
		return
	}
	defer rows.Close()

	var quartiers []gin.H
	for rows.Next() {
		var quartier models.Quartier
		var nameAr sql.NullString
		var createdAt time.Time
		
		err := rows.Scan(&quartier.ID, &quartier.Name, &nameAr, &quartier.IsActive, &createdAt)
		if err != nil {
			continue
		}

		quartierData := gin.H{
			"id":         quartier.ID,
			"name":       quartier.Name,
			"name_ar":    nameAr.String,
			"is_active":  quartier.IsActive,
			"created_at": createdAt,
		}
		quartiers = append(quartiers, quartierData)
	}

	c.JSON(http.StatusOK, gin.H{"quartiers": quartiers})
}

// AdminCreateQuartier creates a new quartier
func AdminCreateQuartier(c *gin.Context) {
	cityID := c.Param("cityId")
	if cityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "City ID is required"})
		return
	}

	var req struct {
		Name     string `json:"name" binding:"required"`
		NameAr   string `json:"name_ar,omitempty"`
		IsActive bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quartierID := uuid.New()
	now := time.Now()
	
	query := `INSERT INTO quartiers (id, city_id, name, name_ar, is_active, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err := DB.Exec(query, quartierID, cityID, req.Name, req.NameAr, req.IsActive, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quartier"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Quartier created successfully",
		"quartier_id": quartierID,
	})
}

// AdminUpdateQuartier updates an existing quartier
func AdminUpdateQuartier(c *gin.Context) {
	quartierID := c.Param("id")
	if quartierID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quartier ID is required"})
		return
	}

	var req struct {
		Name     *string `json:"name,omitempty"`
		NameAr   *string `json:"name_ar,omitempty"`
		IsActive *bool   `json:"is_active,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := "UPDATE quartiers SET "
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		query += "name = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Name)
		argIndex++
	}

	if req.NameAr != nil {
		query += "name_ar = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.NameAr)
		argIndex++
	}

	if req.IsActive != nil {
		query += "is_active = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Add updated_at and WHERE clause
	query += "updated_at = $" + strconv.Itoa(argIndex) + " WHERE id = $" + strconv.Itoa(argIndex+1)
	args = append(args, time.Now(), quartierID)

	_, err := DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quartier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quartier updated successfully"})
}

// AdminDeleteQuartier deletes a quartier (soft delete)
func AdminDeleteQuartier(c *gin.Context) {
	quartierID := c.Param("id")
	if quartierID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quartier ID is required"})
		return
	}

	_, err := DB.Exec("UPDATE quartiers SET is_active = false, updated_at = $1 WHERE id = $2", time.Now(), quartierID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete quartier"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quartier deleted successfully"})
}
