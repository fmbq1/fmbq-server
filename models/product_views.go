package models

import (
	"time"

	"github.com/google/uuid"
)

// ProductView represents a single product view event
type ProductView struct {
	ID                   uuid.UUID `json:"id"`
	ProductID            uuid.UUID `json:"product_id"`
	UserID               uuid.NullUUID `json:"user_id"`               // Nullable for anonymous users
	AnonymousSessionID   string    `json:"anonymous_session_id"`     // For unauthenticated users
	ViewTimestamp        time.Time `json:"view_timestamp"`
	IPAddress            string    `json:"ip_address"`
	UserAgent            string    `json:"user_agent"`
	CreatedAt            time.Time `json:"created_at"`
}

// ProductViewCount represents aggregated view count for a product
type ProductViewCount struct {
	ProductID    uuid.UUID `json:"product_id"`
	ViewCount    int       `json:"view_count"`
	Product      interface{} `json:"product"` // Will contain full product data
}

// TableName returns the table name for ProductView
func (ProductView) TableName() string {
	return "product_views"
}

// CreateTableSQL returns the SQL to create the product_views table
func (ProductView) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS product_views (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_id UUID NOT NULL REFERENCES product_models(id) ON DELETE CASCADE,
		user_id UUID REFERENCES users(id) ON DELETE SET NULL,
		anonymous_session_id VARCHAR(255),
		view_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		ip_address INET,
		user_agent TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	
	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_product_views_product_id ON product_views(product_id);
	CREATE INDEX IF NOT EXISTS idx_product_views_user_id ON product_views(user_id);
	CREATE INDEX IF NOT EXISTS idx_product_views_anonymous_session ON product_views(anonymous_session_id);
	CREATE INDEX IF NOT EXISTS idx_product_views_timestamp ON product_views(view_timestamp);
	CREATE INDEX IF NOT EXISTS idx_product_views_composite ON product_views(product_id, view_timestamp);
	
	-- Prevent duplicate views from same user/session within 5 minutes
	CREATE UNIQUE INDEX IF NOT EXISTS idx_product_views_unique_user 
	ON product_views(product_id, user_id, view_timestamp) 
	WHERE user_id IS NOT NULL;
	
	CREATE UNIQUE INDEX IF NOT EXISTS idx_product_views_unique_anonymous 
	ON product_views(product_id, anonymous_session_id, view_timestamp) 
	WHERE anonymous_session_id IS NOT NULL;
	`
}
