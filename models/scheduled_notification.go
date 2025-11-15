package models

import (
	"time"

	"github.com/google/uuid"
)

// ScheduledNotification represents a scheduled push notification
type ScheduledNotification struct {
	ID              uuid.UUID `json:"id" db:"id"`
	UserID          uuid.UUID `json:"user_id" db:"user_id"`
	Type            string    `json:"type" db:"type"` // "cart-reminder", "wishlist-reminder"
	ReminderType    string    `json:"reminder_type" db:"reminder_type"` // "6h", "24h", "3d", "weekly"
	ProductID       *uuid.UUID `json:"product_id" db:"product_id"`
	ProductName     string    `json:"product_name" db:"product_name"`
	ProductImageURL string    `json:"product_image_url" db:"product_image_url"`
	ProductPrice    float64   `json:"product_price" db:"product_price"`
	ScheduledFor    time.Time `json:"scheduled_for" db:"scheduled_for"`
	Sent            bool      `json:"sent" db:"sent"`
	Cancelled       bool      `json:"cancelled" db:"cancelled"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

func (ScheduledNotification) TableName() string {
	return "scheduled_notifications"
}

func (ScheduledNotification) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS scheduled_notifications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		type VARCHAR(50) NOT NULL CHECK (type IN ('cart-reminder', 'wishlist-reminder')),
		reminder_type VARCHAR(20) NOT NULL CHECK (reminder_type IN ('6h', '24h', '3d', 'weekly')),
		product_id UUID,
		product_name TEXT NOT NULL,
		product_image_url TEXT NOT NULL,
		product_price DECIMAL(10, 2) DEFAULT 0,
		scheduled_for TIMESTAMP WITH TIME ZONE NOT NULL,
		sent BOOLEAN DEFAULT FALSE,
		cancelled BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

