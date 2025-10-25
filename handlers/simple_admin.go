package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SimpleAdminSignupRequest struct {
	Phone     string `json:"phone" binding:"required"`
	Password  string `json:"password" binding:"required"`
	FullName  string `json:"full_name" binding:"required"`
	Email     string `json:"email"`
}

// SimpleAdminSignup - A simplified admin signup that just works
func SimpleAdminSignup(c *gin.Context) {
	var req SimpleAdminSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password hashing failed"})
		return
	}

	// Simple insert - let the database handle defaults
	userID := uuid.New()
	now := time.Now()
	
	query := `INSERT INTO users (id, phone, email, password_hash, full_name, role, is_active, created_at, metadata) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err = DB.Exec(query, 
		userID, 
		req.Phone, 
		req.Email, 
		string(hashedPassword), 
		req.FullName, 
		"admin", 
		true, 
		now, 
		"{}")
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database error: " + err.Error(),
			"debug": fmt.Sprintf("Phone: %s, Email: %s, FullName: %s", req.Phone, req.Email, req.FullName),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin created successfully!",
		"user_id": userID,
		"phone":   req.Phone,
	})
}

// SimpleAdminLogin - A simplified admin login
func SimpleAdminLogin(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Find user
	var user models.User
	var passwordHash string
	
	query := `SELECT id, phone, email, password_hash, full_name, role, is_active, created_at, metadata
	          FROM users WHERE phone = $1 AND role = 'admin'`
	
	err := DB.QueryRow(query, req.Phone).Scan(
		&user.ID, &user.Phone, &user.Email, &passwordHash, &user.FullName,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.Metadata,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Generate simple token (for now, just return success)
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful!",
		"user": gin.H{
			"id":         user.ID,
			"phone":      user.Phone,
			"full_name":  user.FullName,
			"role":       user.Role,
		},
	})
}
