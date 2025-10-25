package models

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       *uuid.UUID `json:"user_id" db:"user_id"`
	CompanyName  *string    `json:"company_name" db:"company_name"`
	ContactName  *string    `json:"contact_name" db:"contact_name"`
	Email        *string    `json:"email" db:"email"`
	Phone        *string    `json:"phone" db:"phone"`
	Address      *string    `json:"address" db:"address"`
	City         *string    `json:"city" db:"city"`
	State        *string    `json:"state" db:"state"`
	Country      *string    `json:"country" db:"country"`
	PostalCode   *string    `json:"postal_code" db:"postal_code"`
	CustomerType string     `json:"customer_type" db:"customer_type"` // individual, business
	Status       string     `json:"status" db:"status"`               // active, inactive, prospect
	Source       string     `json:"source" db:"source"`               // website, referral, walk-in, etc.
	Tags         string     `json:"tags" db:"tags"`                   // JSON array of tags
	Notes        *string    `json:"notes" db:"notes"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	LastContact  *time.Time `json:"last_contact" db:"last_contact"`
}

func (Customer) TableName() string {
	return "customers"
}

func (Customer) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS customers (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE SET NULL,
		company_name TEXT,
		contact_name TEXT,
		email TEXT,
		phone TEXT,
		address TEXT,
		city TEXT,
		state TEXT,
		country TEXT,
		postal_code TEXT,
		customer_type TEXT DEFAULT 'individual' CHECK (customer_type IN ('individual', 'business')),
		status TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'prospect')),
		source TEXT DEFAULT 'website',
		tags JSONB DEFAULT '[]',
		notes TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		last_contact TIMESTAMP WITH TIME ZONE
	);
	
	CREATE INDEX IF NOT EXISTS idx_customers_user_id ON customers(user_id);
	CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);
	CREATE INDEX IF NOT EXISTS idx_customers_phone ON customers(phone);
	CREATE INDEX IF NOT EXISTS idx_customers_status ON customers(status);
	CREATE INDEX IF NOT EXISTS idx_customers_customer_type ON customers(customer_type);
	`
}
