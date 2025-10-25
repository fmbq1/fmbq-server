package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetCRMStats returns overall CRM statistics
func GetCRMStats(c *gin.Context) {
	// Get total customers
	var totalCustomers int
	totalQuery := `SELECT COUNT(*) FROM customers`
	err := DB.QueryRow(totalQuery).Scan(&totalCustomers)
	if err != nil {
		// If there's an error, return zero stats instead of 500
		c.JSON(http.StatusOK, gin.H{
			"total_customers":     0,
			"active_customers":    0,
			"inactive_customers":  0,
			"prospect_customers":  0,
			"business_customers":  0,
			"individual_customers": 0,
			"recent_interactions": 0,
			"pending_followups":   0,
		})
		return
	}

	// Get active customers
	var activeCustomers int
	activeQuery := `SELECT COUNT(*) FROM customers WHERE status = 'active'`
	err = DB.QueryRow(activeQuery).Scan(&activeCustomers)
	if err != nil {
		activeCustomers = 0
	}

	// Get inactive customers
	var inactiveCustomers int
	inactiveQuery := `SELECT COUNT(*) FROM customers WHERE status = 'inactive'`
	err = DB.QueryRow(inactiveQuery).Scan(&inactiveCustomers)
	if err != nil {
		inactiveCustomers = 0
	}

	// Get prospect customers
	var prospectCustomers int
	prospectQuery := `SELECT COUNT(*) FROM customers WHERE status = 'prospect'`
	err = DB.QueryRow(prospectQuery).Scan(&prospectCustomers)
	if err != nil {
		prospectCustomers = 0
	}

	// Get business customers
	var businessCustomers int
	businessQuery := `SELECT COUNT(*) FROM customers WHERE customer_type = 'business'`
	err = DB.QueryRow(businessQuery).Scan(&businessCustomers)
	if err != nil {
		businessCustomers = 0
	}

	// Get individual customers
	var individualCustomers int
	individualQuery := `SELECT COUNT(*) FROM customers WHERE customer_type = 'individual'`
	err = DB.QueryRow(individualQuery).Scan(&individualCustomers)
	if err != nil {
		individualCustomers = 0
	}

	// Get recent interactions (last 7 days)
	var recentInteractions int
	recentQuery := `SELECT COUNT(*) FROM customer_interactions WHERE created_at >= NOW() - INTERVAL '7 days'`
	err = DB.QueryRow(recentQuery).Scan(&recentInteractions)
	if err != nil {
		recentInteractions = 0
	}

	// Get pending follow-ups
	var pendingFollowups int
	pendingQuery := `SELECT COUNT(*) FROM customer_interactions WHERE status = 'pending' AND follow_up IS NOT NULL AND follow_up <= NOW()`
	err = DB.QueryRow(pendingQuery).Scan(&pendingFollowups)
	if err != nil {
		pendingFollowups = 0
	}

	stats := gin.H{
		"total_customers":     totalCustomers,
		"active_customers":    activeCustomers,
		"inactive_customers":  inactiveCustomers,
		"prospect_customers":  prospectCustomers,
		"business_customers":  businessCustomers,
		"individual_customers": individualCustomers,
		"recent_interactions": recentInteractions,
		"pending_followups":   pendingFollowups,
	}

	c.JSON(http.StatusOK, stats)
}
