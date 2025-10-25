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

// GetCustomerInteractions returns all interactions for a customer
func GetCustomerInteractions(c *gin.Context) {
	customerID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	interactionType := c.Query("type")
	status := c.Query("status")

	_, err := uuid.Parse(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	offset := (page - 1) * limit

	query := `
		SELECT ci.id, ci.customer_id, ci.user_id, ci.type, ci.subject, ci.description, 
		       ci.outcome, ci.priority, ci.status, ci.duration, ci.follow_up, 
		       ci.created_at, ci.updated_at,
		       u.full_name as user_name
		FROM customer_interactions ci
		LEFT JOIN users u ON ci.user_id = u.id
		WHERE ci.customer_id = $1
	`
	args := []interface{}{customerID}
	argIndex := 2

	if interactionType != "" {
		query += ` AND ci.type = $` + strconv.Itoa(argIndex)
		args = append(args, interactionType)
		argIndex++
	}

	if status != "" {
		query += ` AND ci.status = $` + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	query += ` ORDER BY ci.created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch interactions"})
		return
	}
	defer rows.Close()

	var interactions []gin.H
	for rows.Next() {
		var interaction models.CustomerInteraction
		var description, outcome sql.NullString
		var duration sql.NullInt32
		var followUp sql.NullTime
		var userName sql.NullString

		err := rows.Scan(
			&interaction.ID, &interaction.CustomerID, &interaction.UserID, &interaction.Type,
			&interaction.Subject, &description, &outcome, &interaction.Priority,
			&interaction.Status, &duration, &followUp, &interaction.CreatedAt,
			&interaction.UpdatedAt, &userName,
		)
		if err != nil {
			continue
		}

		interactionData := gin.H{
			"id":           interaction.ID,
			"customer_id":  interaction.CustomerID,
			"user_id":      interaction.UserID,
			"type":         interaction.Type,
			"subject":     interaction.Subject,
			"description":  description.String,
			"outcome":      outcome.String,
			"priority":     interaction.Priority,
			"status":       interaction.Status,
			"duration":     duration.Int32,
			"follow_up":    followUp.Time,
			"created_at":   interaction.CreatedAt,
			"updated_at":   interaction.UpdatedAt,
			"user_name":    userName.String,
		}

		interactions = append(interactions, interactionData)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM customer_interactions WHERE customer_id = $1`
	if interactionType != "" {
		countQuery += ` AND type = $2`
		if status != "" {
			countQuery += ` AND status = $3`
		}
	} else if status != "" {
		countQuery += ` AND status = $2`
	}

	var total int
	err = DB.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		total = len(interactions)
	}

	c.JSON(http.StatusOK, gin.H{
		"interactions": interactions,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// CreateCustomerInteraction creates a new interaction
func CreateCustomerInteraction(c *gin.Context) {
	customerID := c.Param("id")

	_, err := uuid.Parse(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var req struct {
		Type        string     `json:"type" binding:"required,oneof=call email meeting support sale follow_up other"`
		Subject     string     `json:"subject" binding:"required"`
		Description *string    `json:"description"`
		Outcome     *string    `json:"outcome"`
		Priority    string     `json:"priority" binding:"oneof=low medium high urgent"`
		Status      string     `json:"status" binding:"oneof=pending completed cancelled"`
		Duration    *int       `json:"duration"`
		FollowUp    *time.Time `json:"follow_up"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Status == "" {
		req.Status = "pending"
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	interactionID := uuid.New()
	now := time.Now()

	query := `
		INSERT INTO customer_interactions (id, customer_id, user_id, type, subject, description, 
		                                  outcome, priority, status, duration, follow_up, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = DB.Exec(query,
		interactionID, customerID, userID, req.Type, req.Subject, req.Description,
		req.Outcome, req.Priority, req.Status, req.Duration, req.FollowUp, now, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create interaction"})
		return
	}

	// Update customer's last_contact
	updateQuery := `UPDATE customers SET last_contact = $1, updated_at = $2 WHERE id = $3`
	_, err = DB.Exec(updateQuery, now, now, customerID)
	if err != nil {
		// Log error but don't fail the interaction creation
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Interaction created successfully",
		"interaction_id": interactionID,
	})
}

// UpdateCustomerInteraction updates an existing interaction
func UpdateCustomerInteraction(c *gin.Context) {
	interactionID := c.Param("id")

	_, err := uuid.Parse(interactionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interaction ID"})
		return
	}

	var req struct {
		Type        *string    `json:"type"`
		Subject     *string    `json:"subject"`
		Description *string    `json:"description"`
		Outcome     *string    `json:"outcome"`
		Priority    *string    `json:"priority"`
		Status      *string    `json:"status"`
		Duration    *int       `json:"duration"`
		FollowUp    *time.Time `json:"follow_up"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if interaction exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM customer_interactions WHERE id = $1)`
	err = DB.QueryRow(checkQuery, interactionID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Interaction not found"})
		return
	}

	// Build dynamic update query
	query := "UPDATE customer_interactions SET "
	args := []interface{}{}
	argIndex := 1

	if req.Type != nil {
		query += "type = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Type)
		argIndex++
	}

	if req.Subject != nil {
		query += "subject = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Subject)
		argIndex++
	}

	if req.Description != nil {
		query += "description = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Description)
		argIndex++
	}

	if req.Outcome != nil {
		query += "outcome = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Outcome)
		argIndex++
	}

	if req.Priority != nil {
		query += "priority = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Priority)
		argIndex++
	}

	if req.Status != nil {
		query += "status = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Status)
		argIndex++
	}

	if req.Duration != nil {
		query += "duration = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Duration)
		argIndex++
	}

	if req.FollowUp != nil {
		query += "follow_up = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.FollowUp)
		argIndex++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2] + ", updated_at = $" + strconv.Itoa(argIndex) + " WHERE id = $" + strconv.Itoa(argIndex+1)
	args = append(args, time.Now(), interactionID)

	_, err = DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update interaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Interaction updated successfully"})
}

// DeleteCustomerInteraction deletes an interaction
func DeleteCustomerInteraction(c *gin.Context) {
	interactionID := c.Param("id")

	_, err := uuid.Parse(interactionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interaction ID"})
		return
	}

	// Check if interaction exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM customer_interactions WHERE id = $1)`
	err = DB.QueryRow(checkQuery, interactionID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Interaction not found"})
		return
	}

	query := `DELETE FROM customer_interactions WHERE id = $1`
	_, err = DB.Exec(query, interactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete interaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Interaction deleted successfully"})
}

// GetCustomerStats returns statistics for a customer
func GetCustomerStats(c *gin.Context) {
	customerID := c.Param("id")

	_, err := uuid.Parse(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	// Get interaction counts by type
	interactionStats := make(map[string]int)
	types := []string{"call", "email", "meeting", "support", "sale", "follow_up", "other"}

	for _, interactionType := range types {
		var count int
		query := `SELECT COUNT(*) FROM customer_interactions WHERE customer_id = $1 AND type = $2`
		DB.QueryRow(query, customerID, interactionType).Scan(&count)
		interactionStats[interactionType] = count
	}

	// Get total interactions
	var totalInteractions int
	query := `SELECT COUNT(*) FROM customer_interactions WHERE customer_id = $1`
	DB.QueryRow(query, customerID).Scan(&totalInteractions)

	// Get pending interactions
	var pendingInteractions int
	query = `SELECT COUNT(*) FROM customer_interactions WHERE customer_id = $1 AND status = 'pending'`
	DB.QueryRow(query, customerID).Scan(&pendingInteractions)

	// Get last interaction
	var lastInteraction sql.NullTime
	query = `SELECT MAX(created_at) FROM customer_interactions WHERE customer_id = $1`
	DB.QueryRow(query, customerID).Scan(&lastInteraction)

	// Get loyalty information if customer has a user account
	var loyaltyInfo gin.H
	query = `
		SELECT la.points_balance, la.tier, la.total_earned, la.total_redeemed, la.last_activity
		FROM customers c
		LEFT JOIN loyalty_accounts la ON c.user_id = la.user_id
		WHERE c.id = $1
	`
	var pointsBalance, totalEarned, totalRedeemed sql.NullInt64
	var tier sql.NullString
	var lastActivity sql.NullTime

	err = DB.QueryRow(query, customerID).Scan(&pointsBalance, &tier, &totalEarned, &totalRedeemed, &lastActivity)
	if err == nil && pointsBalance.Valid {
		loyaltyInfo = gin.H{
			"points_balance": pointsBalance.Int64,
			"tier":          tier.String,
			"total_earned":  totalEarned.Int64,
			"total_redeemed": totalRedeemed.Int64,
			"last_activity": lastActivity.Time,
		}
	}

	stats := gin.H{
		"interaction_stats":    interactionStats,
		"total_interactions":   totalInteractions,
		"pending_interactions": pendingInteractions,
		"last_interaction":     lastInteraction.Time,
		"loyalty":             loyaltyInfo,
	}

	c.JSON(http.StatusOK, stats)
}
