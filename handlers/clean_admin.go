package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type CleanSignupRequest struct {
	Phone     string `json:"phone" binding:"required"`
	Password  string `json:"password" binding:"required"`
	FullName  string `json:"full_name" binding:"required"`
	Email     string `json:"email"`
}

type CleanLoginRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// CleanAdminSignup - Simple, working admin signup
func CleanAdminSignup(c *gin.Context) {
	var req CleanSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Check if phone exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE phone = $1", req.Phone).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Phone already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password error"})
		return
	}

	// Insert user
	userID := uuid.New()
	now := time.Now()
	
	_, err = DB.Exec(`
		INSERT INTO users (id, phone, email, password_hash, full_name, role, is_active, created_at, metadata) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, userID, req.Phone, req.Email, string(hashedPassword), req.FullName, "admin", true, now, "{}")
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin created successfully",
		"user_id": userID,
		"phone":   req.Phone,
	})
}

// CleanAdminLogin - Simple, working admin login
func CleanAdminLogin(c *gin.Context) {
	var req CleanLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get user data - check both phone and email
	var userID, phone, email, fullName, passwordHash string
	var isActive bool
	
	err := DB.QueryRow(`
		SELECT id, phone, email, full_name, password_hash, is_active 
		FROM users 
		WHERE (phone = $1 OR email = $1) AND role = 'admin'
	`, req.Phone).Scan(&userID, &phone, &email, &fullName, &passwordHash, &isActive)
	
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	if !isActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account deactivated"})
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Generate permanent UUID-based token (same as regular user login)
	permanentToken := uuid.New().String()
	
	fmt.Printf("🔑 Generated admin token for user %s: %s (length: %d)\n", userID, permanentToken, len(permanentToken))
	
	// Store token in database
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		fmt.Printf("❌ Failed to parse user ID: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}
	
	tokenID := uuid.New()
	insertTokenQuery := `INSERT INTO user_tokens (id, user_id, token, created_at, last_used, revoked)
	                     VALUES ($1, $2, $3, $4, $5, $6)`
	result, err := DB.Exec(insertTokenQuery,
		tokenID, userUUID, permanentToken, time.Now(), time.Now(), false)
	if err != nil {
		fmt.Printf("❌ Failed to insert admin token: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Admin token inserted successfully. Rows affected: %d\n", rowsAffected)
	
	// Verify token was inserted
	var verifyToken string
	verifyQuery := `SELECT token FROM user_tokens WHERE token = $1 AND revoked = false`
	err = DB.QueryRow(verifyQuery, permanentToken).Scan(&verifyToken)
	if err != nil {
		fmt.Printf("⚠️ WARNING: Admin token verification failed after insert: %v\n", err)
	} else {
		fmt.Printf("✅ Admin token verified in database: %s\n", verifyToken[:20] + "...")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   permanentToken,
		"user": gin.H{
			"id":        userID,
			"phone":     phone,
			"email":     email,
			"full_name": fullName,
			"role":      "admin",
		},
	})
}
