package models

import (
	"time"

	"github.com/google/uuid"
)

type Address struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Name       *string   `json:"name" db:"name"`
	Line1      *string   `json:"line1" db:"line1"`
	Line2      *string   `json:"line2" db:"line2"`
	City       *string   `json:"city" db:"city"`
	Region     *string   `json:"region" db:"region"`
	PostalCode *string   `json:"postal_code" db:"postal_code"`
	Country    *string   `json:"country" db:"country"`
	Phone      *string   `json:"phone" db:"phone"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

func (Address) TableName() string {
	return "addresses"
}

func (Address) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS addresses (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		name TEXT,
		line1 TEXT,
		line2 TEXT,
		city TEXT,
		region TEXT,
		postal_code TEXT,
		country CHAR(2),
		phone TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
