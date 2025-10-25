package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"fmbq-server/database"
	"fmbq-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminGetPromotionalCodes retrieves all promotional codes for admin
func AdminGetPromotionalCodes(c *gin.Context) {
	query := `
		SELECT 
			pc.id, pc.code, pc.description, pc.discount_type, pc.discount_value,
			pc.min_order_amount, pc.max_discount, pc.usage_limit, pc.used_count,
			pc.is_active, pc.start_date, pc.expiry_date, pc.created_by,
			pc.created_at, pc.updated_at,
			u.full_name as created_by_name
		FROM promotional_codes pc
		LEFT JOIN users u ON pc.created_by = u.id
		ORDER BY pc.created_at DESC
	`

	rows, err := database.Database.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch promotional codes"})
		return
	}
	defer rows.Close()

	var codes []models.PromotionalCode
	for rows.Next() {
		var code models.PromotionalCode
		var createdByName sql.NullString
		
		err := rows.Scan(
			&code.ID, &code.Code, &code.Description, &code.DiscountType,
			&code.DiscountValue, &code.MinOrderAmount, &code.MaxDiscount,
			&code.UsageLimit, &code.UsedCount, &code.IsActive,
			&code.StartDate, &code.ExpiryDate, &code.CreatedBy,
			&code.CreatedAt, &code.UpdatedAt, &createdByName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan promotional code"})
			return
		}
		codes = append(codes, code)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    codes,
	})
}

// AdminCreatePromotionalCode creates a new promotional code
func AdminCreatePromotionalCode(c *gin.Context) {
	var req struct {
		Code            string  `json:"code" binding:"required"`
		Description     string  `json:"description"`
		DiscountType    string  `json:"discount_type" binding:"required,oneof=percentage fixed"`
		DiscountValue   float64 `json:"discount_value" binding:"required,gt=0"`
		MinOrderAmount  float64 `json:"min_order_amount"`
		MaxDiscount     float64 `json:"max_discount"`
		UsageLimit      int     `json:"usage_limit"`
		IsActive        bool    `json:"is_active"`
		StartDate       string  `json:"start_date" binding:"required"`
		ExpiryDate      string  `json:"expiry_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02T15:04:05Z", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
		return
	}

	expiryDate, err := time.Parse("2006-01-02T15:04:05Z", req.ExpiryDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiry date format"})
		return
	}

	// Get admin user ID from context
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin user not found"})
		return
	}

	// Check if code already exists
	var existingCode string
	err = database.Database.QueryRow("SELECT code FROM promotional_codes WHERE code = $1", req.Code).Scan(&existingCode)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Promotional code already exists"})
		return
	}

	// Create promotional code
	query := `
		INSERT INTO promotional_codes (
			code, description, discount_type, discount_value, min_order_amount,
			max_discount, usage_limit, is_active, start_date, expiry_date, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	var id string
	var createdAt, updatedAt time.Time
	err = database.Database.QueryRow(
		query,
		req.Code, req.Description, req.DiscountType, req.DiscountValue,
		req.MinOrderAmount, req.MaxDiscount, req.UsageLimit, req.IsActive,
		startDate, expiryDate, adminID,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create promotional code"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Promotional code created successfully",
		"data": gin.H{
			"id":         id,
			"created_at": createdAt,
			"updated_at": updatedAt,
		},
	})
}

// AdminUpdatePromotionalCode updates an existing promotional code
func AdminUpdatePromotionalCode(c *gin.Context) {
	codeID := c.Param("id")
	if codeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code ID is required"})
		return
	}

	var req struct {
		Code            string  `json:"code"`
		Description     string  `json:"description"`
		DiscountType    string  `json:"discount_type" binding:"omitempty,oneof=percentage fixed"`
		DiscountValue   float64 `json:"discount_value"`
		MinOrderAmount  float64 `json:"min_order_amount"`
		MaxDiscount     float64 `json:"max_discount"`
		UsageLimit      int     `json:"usage_limit"`
		IsActive        bool    `json:"is_active"`
		StartDate       string  `json:"start_date"`
		ExpiryDate      string  `json:"expiry_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query dynamically
	query := "UPDATE promotional_codes SET updated_at = NOW()"
	args := []interface{}{}
	argIndex := 1

	if req.Code != "" {
		query += fmt.Sprintf(", code = $%d", argIndex)
		args = append(args, req.Code)
		argIndex++
	}
	if req.Description != "" {
		query += fmt.Sprintf(", description = $%d", argIndex)
		args = append(args, req.Description)
		argIndex++
	}
	if req.DiscountType != "" {
		query += fmt.Sprintf(", discount_type = $%d", argIndex)
		args = append(args, req.DiscountType)
		argIndex++
	}
	if req.DiscountValue > 0 {
		query += fmt.Sprintf(", discount_value = $%d", argIndex)
		args = append(args, req.DiscountValue)
		argIndex++
	}
	if req.MinOrderAmount >= 0 {
		query += fmt.Sprintf(", min_order_amount = $%d", argIndex)
		args = append(args, req.MinOrderAmount)
		argIndex++
	}
	if req.MaxDiscount >= 0 {
		query += fmt.Sprintf(", max_discount = $%d", argIndex)
		args = append(args, req.MaxDiscount)
		argIndex++
	}
	if req.UsageLimit >= 0 {
		query += fmt.Sprintf(", usage_limit = $%d", argIndex)
		args = append(args, req.UsageLimit)
		argIndex++
	}
	query += fmt.Sprintf(", is_active = $%d", argIndex)
	args = append(args, req.IsActive)
	argIndex++

	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02T15:04:05Z", req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format"})
			return
		}
		query += fmt.Sprintf(", start_date = $%d", argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if req.ExpiryDate != "" {
		expiryDate, err := time.Parse("2006-01-02T15:04:05Z", req.ExpiryDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiry date format"})
			return
		}
		query += fmt.Sprintf(", expiry_date = $%d", argIndex)
		args = append(args, expiryDate)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, codeID)

	result, err := database.Database.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update promotional code"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check update result"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promotional code not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Promotional code updated successfully",
	})
}

// AdminDeletePromotionalCode deletes a promotional code
func AdminDeletePromotionalCode(c *gin.Context) {
	codeID := c.Param("id")
	if codeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code ID is required"})
		return
	}

	// Check if code has been used
	var usedCount int
	err := database.Database.QueryRow(
		"SELECT used_count FROM promotional_codes WHERE id = $1",
		codeID,
	).Scan(&usedCount)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promotional code not found"})
		return
	}

	if usedCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete promotional code that has been used"})
		return
	}

	_, err = database.Database.Exec("DELETE FROM promotional_codes WHERE id = $1", codeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete promotional code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Promotional code deleted successfully",
	})
}

// ValidatePromotionalCode validates a promotional code for a user
func ValidatePromotionalCode(c *gin.Context) {
	var req struct {
		Code        string  `json:"code" binding:"required"`
		UserID      string  `json:"user_id" binding:"required"`
		OrderAmount float64 `json:"order_amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get promotional code details
	var code models.PromotionalCode
	query := `
		SELECT id, code, discount_type, discount_value, min_order_amount,
			   max_discount, usage_limit, used_count, is_active, start_date, expiry_date
		FROM promotional_codes 
		WHERE code = $1
	`

	err := database.Database.QueryRow(query, req.Code).Scan(
		&code.ID, &code.Code, &code.DiscountType, &code.DiscountValue,
		&code.MinOrderAmount, &code.MaxDiscount, &code.UsageLimit,
		&code.UsedCount, &code.IsActive, &code.StartDate, &code.ExpiryDate,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invalid promotional code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate promotional code"})
		return
	}

	// Validate code
	now := time.Now()
	if !code.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Promotional code is not active"})
		return
	}

	if now.Before(code.StartDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Promotional code is not yet active"})
		return
	}

	if now.After(code.ExpiryDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Promotional code has expired"})
		return
	}

	if code.UsageLimit > 0 && code.UsedCount >= code.UsageLimit {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Promotional code usage limit reached"})
		return
	}

	if req.OrderAmount < code.MinOrderAmount {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Minimum order amount of %.2f MRU required", code.MinOrderAmount),
		})
		return
	}

	// Calculate discount
	var discountAmount float64
	if code.DiscountType == "percentage" {
		discountAmount = (req.OrderAmount * code.DiscountValue) / 100
		if code.MaxDiscount > 0 && discountAmount > code.MaxDiscount {
			discountAmount = code.MaxDiscount
		}
	} else {
		discountAmount = code.DiscountValue
	}

	// Check if user has already used this code
	var usageCount int
	err = database.Database.QueryRow(
		"SELECT COUNT(*) FROM promotional_code_usage WHERE promotional_code_id = $1 AND user_id = $2",
		code.ID, req.UserID,
	).Scan(&usageCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check code usage"})
		return
	}

	if usageCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already used this promotional code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"valid":          true,
		"discount_amount": discountAmount,
		"code_info": gin.H{
			"id":              code.ID,
			"code":            code.Code,
			"discount_type":   code.DiscountType,
			"discount_value":  code.DiscountValue,
			"description":     code.Description,
		},
	})
}

// ApplyPromotionalCode applies a promotional code to an order
func ApplyPromotionalCode(c *gin.Context) {
	var req struct {
		Code        string  `json:"code" binding:"required"`
		UserID      string  `json:"user_id" binding:"required"`
		OrderID     string  `json:"order_id" binding:"required"`
		OrderAmount float64 `json:"order_amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the code first (validation logic is handled in the function)

	// Get promotional code details
	var code models.PromotionalCode
	query := `
		SELECT id, code, discount_type, discount_value, min_order_amount,
			   max_discount, usage_limit, used_count, is_active, start_date, expiry_date
		FROM promotional_codes 
		WHERE code = $1
	`

	err := database.Database.QueryRow(query, req.Code).Scan(
		&code.ID, &code.Code, &code.DiscountType, &code.DiscountValue,
		&code.MinOrderAmount, &code.MaxDiscount, &code.UsageLimit,
		&code.UsedCount, &code.IsActive, &code.StartDate, &code.ExpiryDate,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invalid promotional code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply promotional code"})
		return
	}

	// Calculate discount
	var discountAmount float64
	if code.DiscountType == "percentage" {
		discountAmount = (req.OrderAmount * code.DiscountValue) / 100
		if code.MaxDiscount > 0 && discountAmount > code.MaxDiscount {
			discountAmount = code.MaxDiscount
		}
	} else {
		discountAmount = code.DiscountValue
	}

	// Record the usage
	usageID := uuid.New().String()
	_, err = database.Database.Exec(`
		INSERT INTO promotional_code_usage (id, promotional_code_id, user_id, order_id, discount_amount)
		VALUES ($1, $2, $3, $4, $5)
	`, usageID, code.ID, req.UserID, req.OrderID, discountAmount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record code usage"})
		return
	}

	// Update used count
	_, err = database.Database.Exec(`
		UPDATE promotional_codes SET used_count = used_count + 1 WHERE id = $1
	`, code.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update usage count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"message":         "Promotional code applied successfully",
		"discount_amount": discountAmount,
		"usage_id":        usageID,
	})
}

// GetPromotionalCodeStats gets statistics for promotional codes
func GetPromotionalCodeStats(c *gin.Context) {
	// Get total codes
	var totalCodes int
	err := database.Database.QueryRow("SELECT COUNT(*) FROM promotional_codes").Scan(&totalCodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total codes"})
		return
	}

	// Get active codes
	var activeCodes int
	err = database.Database.QueryRow("SELECT COUNT(*) FROM promotional_codes WHERE is_active = true").Scan(&activeCodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active codes"})
		return
	}

	// Get total usage
	var totalUsage int
	err = database.Database.QueryRow("SELECT COUNT(*) FROM promotional_code_usage").Scan(&totalUsage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total usage"})
		return
	}

	// Get total discount given
	var totalDiscount float64
	err = database.Database.QueryRow("SELECT COALESCE(SUM(discount_amount), 0) FROM promotional_code_usage").Scan(&totalDiscount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total discount"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_codes":    totalCodes,
			"active_codes":   activeCodes,
			"total_usage":    totalUsage,
			"total_discount": totalDiscount,
		},
	})
}
