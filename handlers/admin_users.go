package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetAllUsers returns all users (admin only)
func GetAllUsers(c *gin.Context) {
	query := `SELECT id, email, phone, full_name, role, is_active, created_at, metadata 
	          FROM users ORDER BY created_at DESC`
	
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.Phone, &user.FullName, 
			&user.Role, &user.IsActive, &user.CreatedAt, &user.Metadata,
		)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// GetUserByID returns a specific user by ID (admin only)
func GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	query := `SELECT id, email, phone, full_name, role, is_active, created_at, metadata 
	          FROM users WHERE id = $1`
	
	err = DB.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FullName, 
		&user.Role, &user.IsActive, &user.CreatedAt, &user.Metadata,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateUserRole updates a user's role (admin only)
func UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin employee user"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	err = DB.QueryRow(checkQuery, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update user role
	updateQuery := `UPDATE users SET role = $1 WHERE id = $2`
	_, err = DB.Exec(updateQuery, req.Role, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User role updated successfully"})
}

// ToggleUserStatus toggles user active status (admin only)
func ToggleUserStatus(c *gin.Context) {
	userID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	err = DB.QueryRow(checkQuery, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update user status
	updateQuery := `UPDATE users SET is_active = $1 WHERE id = $2`
	_, err = DB.Exec(updateQuery, req.IsActive, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	status := "activated"
	if !req.IsActive {
		status = "deactivated"
	}

	c.JSON(http.StatusOK, gin.H{"message": "User " + status + " successfully"})
}

// UpdateUserProfileAdmin updates user profile information (admin only)
func UpdateUserProfileAdmin(c *gin.Context) {
	userID := c.Param("id")
	
	// Validate UUID
	_, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		FullName *string `json:"full_name,omitempty"`
		Email    *string `json:"email,omitempty"`
		Phone    *string `json:"phone,omitempty"`
		Metadata *string `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	err = DB.QueryRow(checkQuery, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
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

	if req.Phone != nil {
		query += "phone = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Phone)
		argIndex++
	}

	if req.Metadata != nil {
		query += "metadata = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Metadata)
		argIndex++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2] + " WHERE id = $" + strconv.Itoa(argIndex)
	args = append(args, userID)

	_, err = DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User profile updated successfully"})
}

// GetUsersStats returns user statistics (admin only)
func GetUsersStats(c *gin.Context) {
	// Get total users
	var totalUsers int
	totalQuery := `SELECT COUNT(*) FROM users`
	err := DB.QueryRow(totalQuery).Scan(&totalUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user statistics"})
		return
	}

	// Get active users
	var activeUsers int
	activeQuery := `SELECT COUNT(*) FROM users WHERE is_active = true`
	err = DB.QueryRow(activeQuery).Scan(&activeUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch active user count"})
		return
	}

	// Get users by role
	var roleStats struct {
		Admins    int `json:"admins"`
		Employees int `json:"employees"`
		Users     int `json:"users"`
	}

	adminQuery := `SELECT COUNT(*) FROM users WHERE role = 'admin'`
	DB.QueryRow(adminQuery).Scan(&roleStats.Admins)

	employeeQuery := `SELECT COUNT(*) FROM users WHERE role = 'employee'`
	DB.QueryRow(employeeQuery).Scan(&roleStats.Employees)

	userQuery := `SELECT COUNT(*) FROM users WHERE role = 'user'`
	DB.QueryRow(userQuery).Scan(&roleStats.Users)

	stats := gin.H{
		"total_users":   totalUsers,
		"active_users":  activeUsers,
		"inactive_users": totalUsers - activeUsers,
		"role_stats":    roleStats,
	}

	c.JSON(http.StatusOK, stats)
}
