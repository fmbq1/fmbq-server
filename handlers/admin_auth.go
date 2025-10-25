package handlers

import (
	"net/http"
	"time"

	"fmbq-server/models"
	"fmbq-server/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AdminSignupRequest struct {
	Phone     string `json:"phone" binding:"required"`
	Password  string `json:"password" binding:"required"`
	FullName  string `json:"full_name" binding:"required"`
	Email     string `json:"email"`
}

type AdminLoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AdminSignup handles admin account creation with password
func AdminSignup(c *gin.Context) {
	var req AdminSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if phone number already exists
	var existingUserID string
	err := DB.QueryRow("SELECT id FROM users WHERE phone = $1", req.Phone).Scan(&existingUserID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Phone number already registered"})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	passwordHash := string(hashedPassword)

	// Create the admin user
	// Generate random avatar
	avatarURL := utils.GenerateRandomAvatar()
	
	adminUser := models.User{
		ID:           uuid.New(),
		Phone:        &req.Phone,
		Email:        &req.Email,
		PasswordHash: &passwordHash,
		FullName:     &req.FullName,
		Avatar:       &avatarURL,
		Role:         "admin",
		IsActive:     true,
		CreatedAt:    time.Now(),
		Metadata:     "{}",
	}

	insertQuery := `INSERT INTO users (id, phone, email, password_hash, full_name, avatar, role, is_active, created_at, metadata)
	                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = DB.Exec(insertQuery,
		adminUser.ID, adminUser.Phone, adminUser.Email, adminUser.PasswordHash, adminUser.FullName,
		adminUser.Avatar, adminUser.Role, adminUser.IsActive, adminUser.CreatedAt, adminUser.Metadata,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin user: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin account created successfully",
		"user_id": adminUser.ID,
		"phone":   adminUser.Phone,
		"role":    adminUser.Role,
	})
}

// AdminLogin handles admin login with password
func AdminLogin(c *gin.Context) {
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Find user by phone - simple approach
	var userID string
	var phone, email, fullName, role, passwordHash string
	var isActive bool
	var createdAt time.Time
	var metadata string
	
	query := `SELECT id, phone, email, full_name, role, is_active, password_hash, created_at, metadata
	          FROM users WHERE phone = $1`
	
	err := DB.QueryRow(query, req.Phone).Scan(
		&userID, &phone, &email, &fullName, &role, &isActive, &passwordHash, &createdAt, &metadata,
	)
	
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Check if user is admin
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Check if user is active
	if !isActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is deactivated"})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Generate JWT token
	token, err := generateJWT(userID, phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":         userID,
			"phone":      phone,
			"email":      email,
			"full_name":  fullName,
			"role":       role,
			"is_active":  isActive,
			"created_at": createdAt,
		},
	})
}
