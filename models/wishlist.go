package models

import (
	"time"
)

// WishlistItem represents a product in user's wishlist
type WishlistItem struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	ProductID string    `json:"product_id" db:"product_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// WishlistItemWithProduct represents a wishlist item with product details
type WishlistItemWithProduct struct {
	WishlistItem
	Product interface{} `json:"product"`
}

// TableName returns the table name for WishlistItem
func (w *WishlistItem) TableName() string {
	return "wishlist_items"
}
