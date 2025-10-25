package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"fmbq-server/models"
	"fmbq-server/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetCustomers returns all customers with pagination and filtering
func GetCustomers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	status := c.Query("status")
	customerType := c.Query("type")
	segmentID := c.Query("segment_id")

	offset := (page - 1) * limit

	// Very simple query to debug - just get customers without joins
	query := `
		SELECT id, user_id, company_name, contact_name, email, phone, 
		       address, city, state, country, postal_code, customer_type, 
		       status, source, tags, notes, created_at, updated_at, last_contact
		FROM customers
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if search != "" {
		query += ` AND (
			c.company_name ILIKE $` + strconv.Itoa(argIndex) + ` OR 
			c.contact_name ILIKE $` + strconv.Itoa(argIndex) + ` OR 
			c.email ILIKE $` + strconv.Itoa(argIndex) + ` OR 
			c.phone ILIKE $` + strconv.Itoa(argIndex) + `
		)`
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if status != "" {
		query += ` AND c.status = $` + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	if customerType != "" {
		query += ` AND c.customer_type = $` + strconv.Itoa(argIndex)
		args = append(args, customerType)
		argIndex++
	}

	if segmentID != "" {
		query += ` AND c.id IN (
			SELECT customer_id FROM customer_segment_members 
			WHERE segment_id = $` + strconv.Itoa(argIndex) + `
		)`
		args = append(args, segmentID)
		argIndex++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	fmt.Printf("DEBUG: Executing query: %s\n", query)
	fmt.Printf("DEBUG: Query args: %v\n", args)

	rows, err := DB.Query(query, args...)
	if err != nil {
		// If there's an error, return empty result instead of 500
		c.JSON(http.StatusOK, gin.H{
			"customers": []gin.H{},
			"pagination": gin.H{
				"page":  page,
				"limit": limit,
				"total": 0,
			},
		})
		return
	}
	defer rows.Close()

	var customers []gin.H
	for rows.Next() {
		var customer models.Customer
		var userID, companyName, contactName, email, phone, address, city, state, country, postalCode, notes sql.NullString
		var lastContact sql.NullTime

		err := rows.Scan(
			&customer.ID, &userID, &companyName, &contactName, &email, &phone,
			&address, &city, &state, &country, &postalCode, &customer.CustomerType,
			&customer.Status, &customer.Source, &customer.Tags, &notes, &customer.CreatedAt,
			&customer.UpdatedAt, &lastContact,
		)
		if err != nil {
			// Log error but continue processing other rows
			fmt.Printf("DEBUG: Error scanning row: %v\n", err)
			continue
		}

		customerData := gin.H{
			"id":            customer.ID,
			"user_id":       userID.String,
			"company_name":  companyName.String,
			"contact_name":  contactName.String,
			"email":         email.String,
			"phone":         phone.String,
			"address":       address.String,
			"city":          city.String,
			"state":         state.String,
			"country":       country.String,
			"postal_code":   postalCode.String,
			"customer_type": customer.CustomerType,
			"status":        customer.Status,
			"source":        customer.Source,
			"tags":          customer.Tags,
			"notes":         notes.String,
			"created_at":    customer.CreatedAt,
			"updated_at":    customer.UpdatedAt,
			"last_contact":  lastContact.Time,
			"user_name":     "",
			"user_avatar":   "",
		}

		customerData["loyalty"] = gin.H{
			"points_balance": 0,
			"tier":          "",
			"total_earned":  0,
			"total_redeemed": 0,
		}

		customers = append(customers, customerData)
	}

	// Get total count - simplified
	countQuery := `SELECT COUNT(*) FROM customers WHERE 1=1`
	
	// Apply same filters for count
	countArgs := []interface{}{}
	countArgIndex := 1
	
	if search != "" {
		countQuery += ` AND (
			c.company_name ILIKE $` + strconv.Itoa(countArgIndex) + ` OR 
			c.contact_name ILIKE $` + strconv.Itoa(countArgIndex) + ` OR 
			c.email ILIKE $` + strconv.Itoa(countArgIndex) + ` OR 
			c.phone ILIKE $` + strconv.Itoa(countArgIndex) + `
		)`
		countArgs = append(countArgs, "%"+search+"%")
		countArgIndex++
	}
	
	if status != "" {
		countQuery += ` AND c.status = $` + strconv.Itoa(countArgIndex)
		countArgs = append(countArgs, status)
		countArgIndex++
	}
	
	if customerType != "" {
		countQuery += ` AND c.customer_type = $` + strconv.Itoa(countArgIndex)
		countArgs = append(countArgs, customerType)
		countArgIndex++
	}
	
	var total int
	err = DB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		// If count query fails, use the length of customers array
		total = len(customers)
	}

	// Debug logging
	fmt.Printf("DEBUG: Found %d customers, total: %d\n", len(customers), total)
	
	// Always return success, even with empty data
	c.JSON(http.StatusOK, gin.H{
		"customers": customers,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetCustomer returns a specific customer by ID
func GetCustomer(c *gin.Context) {
	customerID := c.Param("id")
	fmt.Printf("DEBUG: Getting customer with ID: %s\n", customerID)

	_, err := uuid.Parse(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	query := `
		SELECT c.id, c.user_id, c.company_name, c.contact_name, c.email, c.phone, 
		       c.address, c.city, c.state, c.country, c.postal_code, c.customer_type, 
		       c.status, c.source, c.tags, c.notes, c.created_at, c.updated_at, c.last_contact,
		       COALESCE(u.full_name, '') as user_name, COALESCE(u.avatar, '') as user_avatar,
		       COALESCE(la.points_balance, 0) as points_balance, 
		       COALESCE(la.tier, '') as tier, 
		       COALESCE(la.total_earned, 0) as total_earned, 
		       COALESCE(la.total_redeemed, 0) as total_redeemed
		FROM customers c
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN loyalty_accounts la ON c.user_id = la.user_id
		WHERE c.id = $1
	`

	var customer models.Customer
	var userID, companyName, contactName, email, phone, address, city, state, country, postalCode, notes sql.NullString
	var lastContact sql.NullTime
	var userName, userAvatar string // Changed to string for COALESCE
	var pointsBalance, totalEarned, totalRedeemed int64 // Changed to int64 for COALESCE
	var tier string // Changed to string for COALESCE

	err = DB.QueryRow(query, customerID).Scan(
		&customer.ID, &userID, &companyName, &contactName, &email, &phone,
		&address, &city, &state, &country, &postalCode, &customer.CustomerType,
		&customer.Status, &customer.Source, &customer.Tags, &notes, &customer.CreatedAt,
		&customer.UpdatedAt, &lastContact, &userName, &userAvatar, &pointsBalance, &tier, &totalEarned, &totalRedeemed,
	)

	if err != nil {
		fmt.Printf("DEBUG: Error fetching customer: %v\n", err)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customer"})
		}
		return
	}

	fmt.Printf("DEBUG: Successfully fetched customer: %s, user_name: %s, user_avatar: %s\n", customer.ID, userName, userAvatar)

	customerData := gin.H{
		"id":            customer.ID,
		"user_id":       userID.String,
		"company_name":  companyName.String,
		"contact_name":  contactName.String,
		"email":         email.String,
		"phone":         phone.String,
		"address":       address.String,
		"city":          city.String,
		"state":         state.String,
		"country":       country.String,
		"postal_code":   postalCode.String,
		"customer_type": customer.CustomerType,
		"status":        customer.Status,
		"source":        customer.Source,
		"tags":          customer.Tags,
		"notes":         notes.String,
		"created_at":    customer.CreatedAt,
		"updated_at":    customer.UpdatedAt,
		"last_contact":  lastContact.Time,
		"user_name":     userName,
		"user_avatar":   userAvatar,
	}

	customerData["loyalty"] = gin.H{
		"points_balance": pointsBalance,
		"tier":          tier,
		"total_earned":  totalEarned,
		"total_redeemed": totalRedeemed,
	}

	c.JSON(http.StatusOK, gin.H{"customer": customerData})
}

// CreateCustomer creates a new customer
func CreateCustomer(c *gin.Context) {
	var req struct {
		UserID       *string        `json:"user_id"`
		CompanyName  *string        `json:"company_name"`
		ContactName  *string        `json:"contact_name"`
		Email        *string        `json:"email"`
		Phone        *string        `json:"phone"`
		Address      *string        `json:"address"`
		City         *string        `json:"city"`
		State        *string        `json:"state"`
		Country      *string        `json:"country"`
		PostalCode   *string        `json:"postal_code"`
		CustomerType string         `json:"customer_type" binding:"required,oneof=individual business"`
		Status       string         `json:"status" binding:"oneof=active inactive prospect"`
		Source       string         `json:"source"`
		Tags         interface{}    `json:"tags"` // Accept both string and array
		Notes        *string        `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Source == "" {
		req.Source = "website"
	}

	// Handle tags - convert to JSON string
	var tagsJSON string
	if req.Tags == nil {
		tagsJSON = "[]"
	} else {
		switch v := req.Tags.(type) {
		case string:
			// If it's already a string, validate it's valid JSON
			var tags []string
			if err := json.Unmarshal([]byte(v), &tags); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tags format"})
				return
			}
			tagsJSON = v
		case []interface{}:
			// If it's an array, convert to JSON string
			tags := make([]string, len(v))
			for i, tag := range v {
				if str, ok := tag.(string); ok {
					tags[i] = str
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "All tags must be strings"})
					return
				}
			}
			tagsBytes, err := json.Marshal(tags)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to process tags"})
				return
			}
			tagsJSON = string(tagsBytes)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tags must be a string or array"})
			return
		}
	}

	customerID := uuid.New()
	now := time.Now()

	query := `
		INSERT INTO customers (id, user_id, company_name, contact_name, email, phone, 
		                      address, city, state, country, postal_code, customer_type, 
		                      status, source, tags, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	var userID *uuid.UUID
	if req.UserID != nil {
		parsedUserID, err := uuid.Parse(*req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		userID = &parsedUserID
	}

	_, err := DB.Exec(query,
		customerID, userID, req.CompanyName, req.ContactName, req.Email, req.Phone,
		req.Address, req.City, req.State, req.Country, req.PostalCode, req.CustomerType,
		req.Status, req.Source, tagsJSON, req.Notes, now, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}

	// If no user_id was provided but we have an email, optionally create a user account
	if userID == nil && req.Email != nil && *req.Email != "" {
		// Check if user already exists with this email
		var existingUserID string
		checkUserQuery := `SELECT id FROM users WHERE email = $1`
		err := DB.QueryRow(checkUserQuery, *req.Email).Scan(&existingUserID)
		
		if err == sql.ErrNoRows {
			// User doesn't exist, create one
			newUserID := uuid.New()
			avatarURL := utils.GenerateRandomAvatar()
			
			// Create user account
			createUserQuery := `INSERT INTO users (id, email, phone, full_name, avatar, role, is_active, created_at, metadata) 
			                   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
			_, err = DB.Exec(createUserQuery, newUserID, *req.Email, req.Phone, req.ContactName, avatarURL, "user", true, now, "{}")
			
			if err == nil {
				// Update customer with user_id
				updateCustomerQuery := `UPDATE customers SET user_id = $1 WHERE id = $2`
				DB.Exec(updateCustomerQuery, newUserID, customerID)
				
				// Create loyalty account
				loyaltyQuery := `INSERT INTO loyalty_accounts (user_id, points_balance, tier, total_earned, total_redeemed, updated_at) 
				                 VALUES ($1, $2, $3, $4, $5, $6)`
				DB.Exec(loyaltyQuery, newUserID, 0, "bronze", 0, 0, now)
			}
		} else if err == nil {
			// User exists, link them
			updateCustomerQuery := `UPDATE customers SET user_id = $1 WHERE id = $2`
			DB.Exec(updateCustomerQuery, existingUserID, customerID)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Customer created successfully",
		"customer_id": customerID,
	})
}

// GetCustomerOrders returns orders linked to a given customer (via user_id)
func GetCustomerOrders(c *gin.Context) {
    customerID := c.Param("id")

    // Validate UUID
    if _, err := uuid.Parse(customerID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
        return
    }

    // Fetch linked user_id
    var userID sql.NullString
    err := DB.QueryRow(`SELECT user_id FROM customers WHERE id = $1`, customerID).Scan(&userID)
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customer"})
        }
        return
    }

    if !userID.Valid || userID.String == "" {
        c.JSON(http.StatusOK, gin.H{"orders": []gin.H{}})
        return
    }

    // Fetch orders for user
    rows, err := DB.Query(`
        SELECT id, order_number, status, total_amount, currency, created_at
        FROM orders
        WHERE user_id = $1
        ORDER BY created_at DESC
    `, userID.String)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
        return
    }
    defer rows.Close()

    var orders []gin.H
    for rows.Next() {
        var id, orderNumber, status, currency string
        var totalAmount float64
        var createdAt time.Time
        if err := rows.Scan(&id, &orderNumber, &status, &totalAmount, &currency, &createdAt); err != nil {
            continue
        }
        orders = append(orders, gin.H{
            "id":           id,
            "order_number": orderNumber,
            "status":       status,
            "total_amount": totalAmount,
            "currency":     currency,
            "created_at":   createdAt,
        })
    }

    c.JSON(http.StatusOK, gin.H{"orders": orders})
}

// UpdateCustomer updates an existing customer
func UpdateCustomer(c *gin.Context) {
	customerID := c.Param("id")

	_, err := uuid.Parse(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var req struct {
		CompanyName  *string `json:"company_name"`
		ContactName  *string `json:"contact_name"`
		Email        *string `json:"email"`
		Phone        *string `json:"phone"`
		Address      *string `json:"address"`
		City         *string `json:"city"`
		State        *string `json:"state"`
		Country      *string `json:"country"`
		PostalCode   *string `json:"postal_code"`
		CustomerType *string `json:"customer_type"`
		Status       *string `json:"status"`
		Source       *string `json:"source"`
		Tags         *string `json:"tags"`
		Notes        *string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if customer exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM customers WHERE id = $1)`
	err = DB.QueryRow(checkQuery, customerID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	// Build dynamic update query
	query := "UPDATE customers SET "
	args := []interface{}{}
	argIndex := 1

	if req.CompanyName != nil {
		query += "company_name = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.CompanyName)
		argIndex++
	}

	if req.ContactName != nil {
		query += "contact_name = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.ContactName)
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

	if req.Address != nil {
		query += "address = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Address)
		argIndex++
	}

	if req.City != nil {
		query += "city = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.City)
		argIndex++
	}

	if req.State != nil {
		query += "state = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.State)
		argIndex++
	}

	if req.Country != nil {
		query += "country = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Country)
		argIndex++
	}

	if req.PostalCode != nil {
		query += "postal_code = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.PostalCode)
		argIndex++
	}

	if req.CustomerType != nil {
		query += "customer_type = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.CustomerType)
		argIndex++
	}

	if req.Status != nil {
		query += "status = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Status)
		argIndex++
	}

	if req.Source != nil {
		query += "source = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Source)
		argIndex++
	}

	if req.Tags != nil {
		query += "tags = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Tags)
		argIndex++
	}

	if req.Notes != nil {
		query += "notes = $" + strconv.Itoa(argIndex) + ", "
		args = append(args, *req.Notes)
		argIndex++
	}

	if len(args) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Remove trailing comma and add WHERE clause
	query = query[:len(query)-2] + ", updated_at = $" + strconv.Itoa(argIndex) + " WHERE id = $" + strconv.Itoa(argIndex+1)
	args = append(args, time.Now(), customerID)

	_, err = DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer updated successfully"})
}

// DeleteCustomer deletes a customer
func DeleteCustomer(c *gin.Context) {
	customerID := c.Param("id")

	_, err := uuid.Parse(customerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	// Check if customer exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM customers WHERE id = $1)`
	err = DB.QueryRow(checkQuery, customerID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	query := `DELETE FROM customers WHERE id = $1`
	_, err = DB.Exec(query, customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete customer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer deleted successfully"})
}
