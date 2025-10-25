package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetPaymentMethods returns all payment methods
func GetPaymentMethods(c *gin.Context) {
	query := `SELECT id, name, label, description, logo, is_active, created_at, updated_at 
	          FROM payment_methods ORDER BY name`
	
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment methods"})
		return
	}
	defer rows.Close()

	var paymentMethods []gin.H
	for rows.Next() {
		var pm models.PaymentMethod
		var description, logo sql.NullString

		err := rows.Scan(
			&pm.ID, &pm.Name, &pm.Label, &description, &logo, 
			&pm.IsActive, &pm.CreatedAt, &pm.UpdatedAt,
		)
		if err != nil {
			continue
		}

		paymentMethodData := gin.H{
			"id":         pm.ID,
			"name":       pm.Name,
			"label":      pm.Label,
			"description": description.String,
			"logo":       logo.String,
			"is_active":  pm.IsActive,
			"created_at": pm.CreatedAt,
			"updated_at": pm.UpdatedAt,
		}

		paymentMethods = append(paymentMethods, paymentMethodData)
	}

	c.JSON(http.StatusOK, gin.H{"payment_methods": paymentMethods})
}

// GetPaymentMethod returns a specific payment method by ID
func GetPaymentMethod(c *gin.Context) {
	paymentMethodID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(paymentMethodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment method ID"})
		return
	}

	var pm models.PaymentMethod
	var description, logo sql.NullString
	query := `SELECT id, name, label, description, logo, is_active, created_at, updated_at 
	          FROM payment_methods WHERE id = $1`
	
	err = DB.QueryRow(query, paymentMethodID).Scan(
		&pm.ID, &pm.Name, &pm.Label, &description, &logo, 
		&pm.IsActive, &pm.CreatedAt, &pm.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment method"})
		}
		return
	}

	paymentMethodData := gin.H{
		"id":         pm.ID,
		"name":       pm.Name,
		"label":      pm.Label,
		"description": description.String,
		"logo":       logo.String,
		"is_active":  pm.IsActive,
		"created_at": pm.CreatedAt,
		"updated_at": pm.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"payment_method": paymentMethodData})
}

// CreatePaymentMethod creates a new payment method
func CreatePaymentMethod(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Label       string `json:"label" binding:"required"`
		Description string `json:"description,omitempty"`
		Logo        string `json:"logo,omitempty"`
		IsActive    bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paymentMethodID := uuid.New()
	now := time.Now()

	// Prepare values for insertion
	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	
	var logo *string
	if req.Logo != "" {
		logo = &req.Logo
	}

	query := `INSERT INTO payment_methods (id, name, label, description, logo, is_active, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	_, err := DB.Exec(query, paymentMethodID, req.Name, req.Label, description, logo, req.IsActive, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment method"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          paymentMethodID,
		"name":        req.Name,
		"label":       req.Label,
		"description": req.Description,
		"logo":        req.Logo,
		"is_active":   req.IsActive,
		"created_at":  now,
		"updated_at":  now,
		"message":     "Payment method created successfully",
	})
}

// UpdatePaymentMethod updates an existing payment method
func UpdatePaymentMethod(c *gin.Context) {
	paymentMethodID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(paymentMethodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment method ID"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Label       string `json:"label" binding:"required"`
		Description string `json:"description,omitempty"`
		Logo        string `json:"logo,omitempty"`
		IsActive    bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if payment method exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM payment_methods WHERE id = $1)`
	err = DB.QueryRow(checkQuery, paymentMethodID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
		return
	}

	// Prepare values for update
	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	
	var logo *string
	if req.Logo != "" {
		logo = &req.Logo
	}

	// Update payment method
	query := `UPDATE payment_methods SET name = $1, label = $2, description = $3, logo = $4, is_active = $5, updated_at = $6 WHERE id = $7`
	_, err = DB.Exec(query, req.Name, req.Label, description, logo, req.IsActive, time.Now(), paymentMethodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment method"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment method updated successfully",
		"payment_method": gin.H{
			"id":          paymentMethodID,
			"name":        req.Name,
			"label":       req.Label,
			"description": req.Description,
			"logo":        req.Logo,
			"is_active":   req.IsActive,
		},
	})
}

// DeletePaymentMethod deletes a payment method
func DeletePaymentMethod(c *gin.Context) {
	paymentMethodID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(paymentMethodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment method ID"})
		return
	}

	// Check if payment method exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM payment_methods WHERE id = $1)`
	err = DB.QueryRow(checkQuery, paymentMethodID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
		return
	}

	// Delete payment method
	query := `DELETE FROM payment_methods WHERE id = $1`
	_, err = DB.Exec(query, paymentMethodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment method"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment method deleted successfully"})
}

// TogglePaymentMethodStatus toggles payment method active status
func TogglePaymentMethodStatus(c *gin.Context) {
	paymentMethodID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(paymentMethodID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment method ID"})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if payment method exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM payment_methods WHERE id = $1)`
	err = DB.QueryRow(checkQuery, paymentMethodID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
		return
	}

	// Update payment method status
	updateQuery := `UPDATE payment_methods SET is_active = $1, updated_at = $2 WHERE id = $3`
	_, err = DB.Exec(updateQuery, req.IsActive, time.Now(), paymentMethodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment method status"})
		return
	}

	status := "activated"
	if !req.IsActive {
		status = "deactivated"
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment method " + status + " successfully"})
}
