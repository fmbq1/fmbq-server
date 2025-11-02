package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"fmbq-server/database"
	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Check if user exists by phone number
func CheckUserExists(c *gin.Context) {
	phone := c.Query("phone")
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number is required"})
		return
	}

	// Validate phone number (8 digits)
	if len(phone) != 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
		return
	}

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)`
	err := database.Database.QueryRow(query, phone).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exists": exists,
		"phone":  phone,
	})
}

// User login
func LoginUser(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate phone number
	if len(req.Phone) != 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
		return
	}

	// Find user by phone
	var user models.User
	query := `SELECT id, phone, full_name, password_hash, is_active, created_at 
	          FROM users WHERE phone = $1`
	err := database.Database.QueryRow(query, req.Phone).Scan(
		&user.ID, &user.Phone, &user.FullName, &user.PasswordHash, 
		&user.IsActive, &user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is deactivated"})
		return
	}

	// Verify password
	if user.PasswordHash == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Generate JWT token (15 days expiration)
	phoneStr := ""
	if user.Phone != nil {
		phoneStr = *user.Phone
	}
	token, err := generateJWTToken(user.ID.String(), phoneStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Return user data and token
	phoneValue := ""
	if user.Phone != nil {
		phoneValue = *user.Phone
	}
	fullNameValue := ""
	if user.FullName != nil {
		fullNameValue = *user.FullName
	}
	
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"phone":      phoneValue,
			"full_name":  fullNameValue,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
		},
		"token": token,
		"message": "Login successful",
	})
}

// User registration
func RegisterUser(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate phone number
	if len(req.Phone) != 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
		return
	}

	// Validate password (minimum 6 characters)
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters"})
		return
	}

	// Validate name
	if len(req.Name) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name must be at least 2 characters"})
		return
	}

	// Check if user already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)`
	err := database.Database.QueryRow(checkQuery, req.Phone).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	userID := generateUUID()
	insertQuery := `INSERT INTO users (id, phone, full_name, password_hash, is_active, created_at, metadata) 
	                VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err = database.Database.Exec(insertQuery, 
		userID, req.Phone, req.Name, string(hashedPassword), true, time.Now(), "{}")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate JWT token
	token, err := generateJWTToken(userID, req.Phone)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Return user data and token
	c.JSON(http.StatusCreated, gin.H{
		"user": gin.H{
			"id":         userID,
			"phone":      req.Phone,
			"full_name":  req.Name,
			"is_active":  true,
			"created_at": time.Now(),
		},
		"token": token,
		"message": "Registration successful",
	})
}

// Generate JWT token with 15 days expiration
func generateJWTToken(userID, phone string) (string, error) {
	claims := Claims{
		UserID: userID,
		Phone:  phone,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * 24 * time.Hour)), // 15 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("your-secret-key")) // TODO: Move to environment variable
}

// Verify JWT token
func VerifyToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	tokenString := authHeader[7:] // Remove "Bearer " prefix
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
		return
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("your-secret-key"), nil // TODO: Move to environment variable
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Set user info in context for use in protected routes
	c.Set("user_id", claims.UserID)
	c.Set("user_phone", claims.Phone)
	c.Next()
}

// Logout user (client-side token removal)
func LogoutUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// ValidateToken validates a JWT token
func ValidateToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	tokenString := authHeader[7:] // Remove "Bearer " prefix
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
		return
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("your-secret-key"), nil // TODO: Move to environment variable
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user_id": claims.UserID,
		"phone": claims.Phone,
	})
}

// AuthMiddleware validates JWT tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Printf("AuthMiddleware called for: %s %s\n", c.Request.Method, c.Request.URL.Path)
		
		authHeader := c.GetHeader("Authorization")
		fmt.Printf("Authorization header: %s\n", authHeader)
		
		if authHeader == "" {
			fmt.Println("No authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader[7:] // Remove "Bearer " prefix
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			fmt.Println("Invalid authorization format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		fmt.Printf("Token string: %s\n", tokenString)

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("your-secret-key"), nil // TODO: Move to environment variable
		})

		if err != nil {
			fmt.Printf("JWT parse error: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if !token.Valid {
			fmt.Println("Token is not valid")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		fmt.Printf("Token valid, user ID: %s, phone: %s\n", claims.UserID, claims.Phone)

		// Set user info in context for use in protected routes
		c.Set("user_id", claims.UserID)
		c.Set("user_phone", claims.Phone)
		c.Next()
	}
}

// SendOTP placeholder function
func SendOTP(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully"})
}

// VerifyOTP placeholder function
func VerifyOTP(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "OTP verified successfully"})
}

// RefreshToken placeholder function
func RefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Token refreshed successfully"})
}

// Update user push token
func UpdatePushToken(c *gin.Context) {
	fmt.Println("UpdatePushToken called")
	
	var req struct {
		PushToken string `json:"push_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		fmt.Println("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	fmt.Printf("Updating push token for user ID: %v\n", userID)
	fmt.Printf("Push token: %s\n", req.PushToken)

	// If PushToken is empty, set it to NULL to disable notifications
	var pushTokenValue interface{}
	if req.PushToken == "" {
		pushTokenValue = nil
		fmt.Println("Clearing push token (disabling notifications)")
	} else {
		pushTokenValue = req.PushToken
		fmt.Println("Setting push token (enabling notifications)")
	}

	// Update push token in database (NULL if empty string to disable)
	query := `UPDATE users SET push_token = $1, updated_at = now() WHERE id = $2`
	result, err := database.Database.Exec(query, pushTokenValue, userID)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update push token",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Rows affected: %d\n", rowsAffected)

	if req.PushToken == "" {
		c.JSON(http.StatusOK, gin.H{
			"message": "Notifications disabled successfully",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "Push token updated successfully",
		})
	}
}