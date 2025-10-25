package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateAdminUser creates the first admin user (only works if no admin exists)
func CreateAdminUser(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		FullName string `json:"full_name" binding:"required"`
		Email    string `json:"email,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if any admin already exists
	var adminCount int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&adminCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing admins"})
		return
	}

	if adminCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin user already exists"})
		return
	}

	// Create admin user
	adminID := uuid.New()
	now := time.Now()

	insertQuery := `INSERT INTO users (id, phone, full_name, email, role, is_active, created_at, metadata) 
	                VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	
	_, err = DB.Exec(insertQuery, adminID, req.Phone, req.FullName, req.Email, "admin", true, now, "{}")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin user"})
		return
	}

	// Create loyalty account for admin
	loyaltyQuery := `INSERT INTO loyalty_accounts (user_id, points_balance, tier, updated_at) VALUES ($1, $2, $3, $4)`
	_, err = DB.Exec(loyaltyQuery, adminID, 0, "platinum", now)
	if err != nil {
		// Log error but don't fail the admin creation
		println("Failed to create loyalty account for admin:", err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin user created successfully",
		"admin_id": adminID,
		"phone": req.Phone,
		"full_name": req.FullName,
		"email": req.Email,
		"role": "admin",
	})
}
