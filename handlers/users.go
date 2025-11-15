package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var user models.User
	query := `SELECT id, email, phone, full_name, role, is_active, created_at, metadata 
	          FROM users WHERE id = $1`
	
	err := DB.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FullName, 
		&user.Role, &user.IsActive, &user.CreatedAt, &user.Metadata,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user profile"})
		}
		return
	}

	// Get loyalty account info
	var loyaltyAccount models.LoyaltyAccount
	loyaltyQuery := `SELECT user_id, points_balance, tier, updated_at 
	                 FROM loyalty_accounts WHERE user_id = $1`
	
	err = DB.QueryRow(loyaltyQuery, userID).Scan(
		&loyaltyAccount.UserID, &loyaltyAccount.PointsBalance, 
		&loyaltyAccount.Tier, &loyaltyAccount.UpdatedAt,
	)
	
	profile := gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"phone":      user.Phone,
		"full_name":  user.FullName,
		"role":       user.Role,
		"is_active":  user.IsActive,
		"created_at": user.CreatedAt,
		"metadata":   user.Metadata,
	}

	if err == nil {
		profile["loyalty"] = gin.H{
			"points_balance": loyaltyAccount.PointsBalance,
			"tier":          loyaltyAccount.Tier,
			"updated_at":    loyaltyAccount.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, profile)
}

func UpdateUserProfile(c *gin.Context) {
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
		FullName *string `json:"full_name,omitempty"`
		Email    *string `json:"email,omitempty"`
		Metadata string  `json:"metadata,omitempty"`
	}

	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := "UPDATE users SET "
	args := []interface{}{}
	argIndex := 1

	if req.FullName != nil {
		query += "full_name = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.FullName)
		argIndex++
	}

	if req.Email != nil {
		query += "email = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Email)
		argIndex++
	}

	if req.Metadata != "" {
		query += "metadata = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, req.Metadata)
		argIndex++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2] + ", updated_at = now() WHERE id = $" + strconv.Itoa(argIndex)
	args = append(args, userID)

	_, err = DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
		return
	}

	// Return updated user data
	var updatedUser models.User
	fetchQuery := `SELECT id, email, phone, full_name, role, is_active, created_at, metadata 
	               FROM users WHERE id = $1`
	err = DB.QueryRow(fetchQuery, userID).Scan(
		&updatedUser.ID, &updatedUser.Email, &updatedUser.Phone, &updatedUser.FullName,
		&updatedUser.Role, &updatedUser.IsActive, &updatedUser.CreatedAt, &updatedUser.Metadata,
	)
	if err != nil {
		// Still return success even if fetch fails
		c.JSON(http.StatusOK, gin.H{
			"message": "Profile updated successfully",
			"user": gin.H{
				"id":        userID,
				"full_name": req.FullName,
				"email":     req.Email,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user": gin.H{
			"id":        updatedUser.ID,
			"full_name": updatedUser.FullName,
			"email":     updatedUser.Email,
			"phone":     updatedUser.Phone,
		},
	})
}

// func GetUserOrders(c *gin.Context) {
// 	userID, exists := c.Get("user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
// 		return
// 	}

// 	query := `SELECT id, order_number, status, total_amount, currency, 
// 	                 shipping_address_id, billing_address_id, created_at, updated_at
// 	          FROM orders WHERE user_id = $1 ORDER BY created_at DESC`
	
// 	rows, err := DB.Query(query, userID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
// 		return
// 	}
// 	defer rows.Close()

// 	var orders []gin.H
// 	for rows.Next() {
// 		var order models.Order
// 		var shippingAddressID, billingAddressID sql.NullString
		
// 		err := rows.Scan(
// 			&order.ID, &order.OrderNumber, &order.Status, &order.TotalAmount, 
// 			&order.Currency, &shippingAddressID, &billingAddressID, 
// 			&order.CreatedAt, &order.UpdatedAt,
// 		)
// 		if err != nil {
// 			continue
// 		}

// 		orderData := gin.H{
// 			"id":                   order.ID,
// 			"order_number":         order.OrderNumber,
// 			"status":               order.Status,
// 			"total_amount":         order.TotalAmount,
// 			"currency":             order.Currency,
// 			"shipping_address_id":  shippingAddressID.String,
// 			"billing_address_id":   billingAddressID.String,
// 			"created_at":           order.CreatedAt,
// 			"updated_at":           order.UpdatedAt,
// 		}
// 		orders = append(orders, orderData)
// 	}

// 	c.JSON(http.StatusOK, gin.H{"orders": orders})
// }
