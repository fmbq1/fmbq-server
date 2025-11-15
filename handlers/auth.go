package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fmbq-server/database"
	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

	// Generate permanent UUID-based token
	permanentToken := uuid.New().String()
	
	fmt.Printf("🔑 Generated token for user %s: %s (length: %d)\n", user.ID.String(), permanentToken, len(permanentToken))
	
	// Store token in database
	tokenID := uuid.New()
	insertTokenQuery := `INSERT INTO user_tokens (id, user_id, token, created_at, last_used, revoked)
	                     VALUES ($1, $2, $3, $4, $5, $6)`
	result, err := database.Database.Exec(insertTokenQuery,
		tokenID, user.ID, permanentToken, time.Now(), time.Now(), false)
	if err != nil {
		fmt.Printf("❌ Failed to insert token: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Token inserted successfully. Rows affected: %d\n", rowsAffected)
	
	// Verify token was inserted
	var verifyToken string
	verifyQuery := `SELECT token FROM user_tokens WHERE token = $1 AND revoked = false`
	err = database.Database.QueryRow(verifyQuery, permanentToken).Scan(&verifyToken)
	if err != nil {
		fmt.Printf("⚠️ WARNING: Token verification failed after insert: %v\n", err)
	} else {
		fmt.Printf("✅ Token verified in database: %s\n", verifyToken[:20] + "...")
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
		"token": permanentToken,
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

	// Generate permanent UUID-based token
	permanentToken := uuid.New().String()
	
	// Store token in database
	tokenID := uuid.New()
	insertTokenQuery := `INSERT INTO user_tokens (id, user_id, token, created_at, last_used, revoked)
	                     VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = database.Database.Exec(insertTokenQuery,
		tokenID, userID, permanentToken, time.Now(), time.Now(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
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
		"token": permanentToken,
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

// Logout user - revoke token in database
func LogoutUser(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	tokenString := authHeader[7:] // Remove "Bearer " prefix

	// Revoke token in database
	revokeQuery := `UPDATE user_tokens SET revoked = true WHERE token = $1 AND revoked = false`
	result, err := database.Database.Exec(revokeQuery, tokenString)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Token might already be revoked or doesn't exist, but still return success
		c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// ValidateToken validates a permanent token from database
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

	// Validate token from database
	var userID uuid.UUID
	var phone sql.NullString
	
	query := `SELECT ut.user_id, u.phone 
	          FROM user_tokens ut
	          JOIN users u ON ut.user_id = u.id
	          WHERE ut.token = $1 AND ut.revoked = false AND u.is_active = true`
	
	err := database.Database.QueryRow(query, tokenString).Scan(&userID, &phone)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or revoked token"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	phoneValue := ""
	if phone.Valid {
		phoneValue = phone.String
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user_id": userID.String(),
		"phone": phoneValue,
	})
}

// AuthMiddleware validates permanent tokens from database
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

		// Trim any whitespace from token
		tokenString = strings.TrimSpace(tokenString)
		
		fmt.Printf("Token string: %s (length: %d)\n", tokenString, len(tokenString))

		// Validate token format (should be UUID format, 36 chars)
		if len(tokenString) == 0 {
			fmt.Println("Empty token string")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is required"})
			c.Abort()
			return
		}

		// Validate token from database
		var userID uuid.UUID
		var phone sql.NullString
		var fullName sql.NullString
		
		query := `SELECT ut.user_id, u.phone, u.full_name 
		          FROM user_tokens ut
		          JOIN users u ON ut.user_id = u.id
		          WHERE ut.token = $1 AND ut.revoked = false AND u.is_active = true`
		
		err := database.Database.QueryRow(query, tokenString).Scan(&userID, &phone, &fullName)
		if err != nil {
			if err == sql.ErrNoRows {
				tokenPreview := tokenString
				if len(tokenString) > 20 {
					tokenPreview = tokenString[:20] + "..."
				}
				fmt.Printf("❌ Token not found in database. Token preview: %s (length: %d)\n", tokenPreview, len(tokenString))
				
				// Debug: Check if token exists at all (without joins)
				var tokenExists bool
				existsQuery := `SELECT EXISTS(SELECT 1 FROM user_tokens WHERE token = $1)`
				existsErr := database.Database.QueryRow(existsQuery, tokenString).Scan(&tokenExists)
				if existsErr == nil && tokenExists {
					fmt.Printf("⚠️ Token exists in user_tokens table but query failed\n")
					// Check if token is revoked
					var revoked bool
					checkRevokedQuery := `SELECT revoked FROM user_tokens WHERE token = $1`
					revokedErr := database.Database.QueryRow(checkRevokedQuery, tokenString).Scan(&revoked)
					if revokedErr == nil {
						if revoked {
							fmt.Printf("⚠️ Token exists but is revoked\n")
							c.JSON(http.StatusUnauthorized, gin.H{
								"error": "Token has been revoked. Please log in again.",
							})
						} else {
							// Token exists and not revoked, check user status
							var userActive bool
							userQuery := `SELECT u.is_active FROM user_tokens ut JOIN users u ON ut.user_id = u.id WHERE ut.token = $1`
							userErr := database.Database.QueryRow(userQuery, tokenString).Scan(&userActive)
							if userErr == nil && !userActive {
								fmt.Printf("⚠️ Token exists but user is inactive\n")
								c.JSON(http.StatusUnauthorized, gin.H{
									"error": "Account is inactive. Please contact support.",
								})
							} else {
								fmt.Printf("⚠️ Token exists but query join failed: %v\n", userErr)
								c.JSON(http.StatusUnauthorized, gin.H{
									"error": "Invalid token. Please log in again.",
								})
							}
						}
					} else {
						fmt.Printf("⚠️ Could not check revocation status: %v\n", revokedErr)
						c.JSON(http.StatusUnauthorized, gin.H{
							"error": "Invalid token. Please log in again.",
						})
					}
				} else {
					// Token doesn't exist at all
					fmt.Printf("⚠️ Token does not exist in database at all\n")
					// Check if it might be a JWT token (old system) - JWTs are typically > 100 chars
					if len(tokenString) > 100 {
						c.JSON(http.StatusUnauthorized, gin.H{
							"error": "Invalid token. Please log out and log in again to refresh your session.",
						})
					} else {
						c.JSON(http.StatusUnauthorized, gin.H{
							"error": "Invalid token. Please log in again.",
						})
					}
				}
			} else {
				fmt.Printf("❌ Database error during token validation: %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			}
			c.Abort()
			return
		}

		// Update last_used timestamp
		updateQuery := `UPDATE user_tokens SET last_used = $1 WHERE token = $2`
		database.Database.Exec(updateQuery, time.Now(), tokenString)

		phoneValue := ""
		if phone.Valid {
			phoneValue = phone.String
		}

		fmt.Printf("Token valid, user ID: %s, phone: %s\n", userID.String(), phoneValue)

		// Set user info in context for use in protected routes
		c.Set("user_id", userID.String())
		c.Set("user_phone", phoneValue)
		c.Next()
	}
}

// ChangePassword changes user password and revokes all existing tokens
func ChangePassword(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate new password
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be at least 6 characters"})
		return
	}

	// Get current password hash
	var currentPasswordHash string
	query := `SELECT password_hash FROM users WHERE id = $1`
	err = database.Database.QueryRow(query, userID).Scan(&currentPasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(req.CurrentPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update password and revoke all tokens in a transaction
	tx, err := database.Database.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Update password
	updateQuery := `UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`
	_, err = tx.Exec(updateQuery, string(hashedPassword), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Revoke all user tokens
	revokeQuery := `UPDATE user_tokens SET revoked = true WHERE user_id = $1 AND revoked = false`
	_, err = tx.Exec(revokeQuery, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke tokens"})
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully. Please log in again.",
	})
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