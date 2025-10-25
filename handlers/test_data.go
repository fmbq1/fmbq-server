package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateTestCustomer creates a test customer for development
func CreateTestCustomer(c *gin.Context) {
	// Create a test customer
	customerID := uuid.New()
	userID := uuid.New()
	
	// First create a test user
	userQuery := `INSERT INTO users (id, email, phone, full_name, avatar, role, is_active, created_at, metadata) 
	              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	avatarURL := "https://api.dicebear.com/7.x/avataaars/svg?seed=test"
	_, err := DB.Exec(userQuery, userID, "test@example.com", "+1234567890", "Test User", avatarURL, "user", true, time.Now(), "{}")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create test user: " + err.Error()})
		return
	}
	
	// Create loyalty account
	loyaltyQuery := `INSERT INTO loyalty_accounts (user_id, points_balance, tier, total_earned, total_redeemed, updated_at) 
	                 VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = DB.Exec(loyaltyQuery, userID, 100, "bronze", 100, 0, time.Now())
	if err != nil {
		// Log error but don't fail
	}
	
	// Create test customer
	customerQuery := `INSERT INTO customers (id, user_id, company_name, contact_name, email, phone, address, city, state, country, customer_type, status, source, tags, notes, created_at, updated_at) 
	                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`
	
	_, err = DB.Exec(customerQuery,
		customerID, userID, "Test Company", "John Doe", "test@example.com", "+1234567890",
		"123 Test St", "Test City", "Test State", "Test Country", "business", "active", "website",
		"[\"test\", \"demo\"]", "This is a test customer", time.Now(), time.Now())
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create test customer: " + err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Test customer created successfully",
		"customer_id": customerID,
		"user_id": userID,
	})
}

// TestGetCustomers returns all customers for debugging
func TestGetCustomers(c *gin.Context) {
	// Simple query to get all customers
	query := `SELECT id, company_name, contact_name, email, customer_type, status, created_at FROM customers ORDER BY created_at DESC`
	
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customers: " + err.Error()})
		return
	}
	defer rows.Close()

	var customers []gin.H
	for rows.Next() {
		var id, companyName, contactName, email, customerType, status, createdAt string
		
		err := rows.Scan(&id, &companyName, &contactName, &email, &customerType, &status, &createdAt)
		if err != nil {
			continue
		}

		customerData := gin.H{
			"id":            id,
			"company_name":  companyName,
			"contact_name":  contactName,
			"email":         email,
			"customer_type": customerType,
			"status":        status,
			"created_at":    createdAt,
		}
		customers = append(customers, customerData)
	}

	c.JSON(http.StatusOK, gin.H{
		"customers": customers,
		"count": len(customers),
	})
}
