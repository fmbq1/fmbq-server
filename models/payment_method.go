package models

import (
	"time"

	"github.com/google/uuid"
)

type PaymentMethod struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Label       string    `json:"label" db:"label"`
	Description *string   `json:"description" db:"description"`
	Logo        *string   `json:"logo" db:"logo"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (PaymentMethod) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS payment_methods (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		label TEXT NOT NULL,
		description TEXT,
		logo TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
