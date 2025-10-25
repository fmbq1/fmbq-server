package models

import (
	"time"
)

// PromotionalCode represents a discount code that can be applied to orders
type PromotionalCode struct {
	ID              string    `json:"id" db:"id"`
	Code            string    `json:"code" db:"code"`
	Description     string    `json:"description" db:"description"`
	DiscountType    string    `json:"discount_type" db:"discount_type"` // "percentage" or "fixed"
	DiscountValue   float64   `json:"discount_value" db:"discount_value"`
	MinOrderAmount  float64   `json:"min_order_amount" db:"min_order_amount"`
	MaxDiscount     float64   `json:"max_discount" db:"max_discount"`
	UsageLimit      int       `json:"usage_limit" db:"usage_limit"`
	UsedCount       int       `json:"used_count" db:"used_count"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	StartDate       time.Time `json:"start_date" db:"start_date"`
	ExpiryDate      time.Time `json:"expiry_date" db:"expiry_date"`
	CreatedBy       string    `json:"created_by" db:"created_by"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// PromotionalCodeUsage tracks when and by whom a code was used
type PromotionalCodeUsage struct {
	ID               string    `json:"id" db:"id"`
	PromotionalCodeID string   `json:"promotional_code_id" db:"promotional_code_id"`
	UserID           string    `json:"user_id" db:"user_id"`
	OrderID          string    `json:"order_id" db:"order_id"`
	DiscountAmount   float64   `json:"discount_amount" db:"discount_amount"`
	UsedAt           time.Time `json:"used_at" db:"used_at"`
}

// TableName returns the table name for PromotionalCode
func (p *PromotionalCode) TableName() string {
	return "promotional_codes"
}

// TableName returns the table name for PromotionalCodeUsage
func (p *PromotionalCodeUsage) TableName() string {
	return "promotional_code_usage"
}

// CreateTableSQL returns the SQL to create the promotional_codes table
func (p *PromotionalCode) CreateTableSQL() string {
	return `
		CREATE TABLE IF NOT EXISTS promotional_codes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			code VARCHAR(50) UNIQUE NOT NULL,
			description TEXT,
			discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('percentage', 'fixed')),
			discount_value DECIMAL(10,2) NOT NULL,
			min_order_amount DECIMAL(10,2) DEFAULT 0,
			max_discount DECIMAL(10,2),
			usage_limit INTEGER DEFAULT -1,
			used_count INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT true,
			start_date TIMESTAMP WITH TIME ZONE NOT NULL,
			expiry_date TIMESTAMP WITH TIME ZONE NOT NULL,
			created_by UUID REFERENCES users(id),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
}

// CreateTableSQL returns the SQL to create the promotional_code_usage table
func (p *PromotionalCodeUsage) CreateTableSQL() string {
	return `
		CREATE TABLE IF NOT EXISTS promotional_code_usage (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			promotional_code_id UUID NOT NULL REFERENCES promotional_codes(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
			discount_amount DECIMAL(10,2) NOT NULL,
			used_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
}
