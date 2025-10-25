package models

import (
	"time"

	"github.com/google/uuid"
)

type CustomerInteraction struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CustomerID  uuid.UUID `json:"customer_id" db:"customer_id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"` // Staff member who handled the interaction
	Type        string    `json:"type" db:"type"`       // call, email, meeting, support, sale, etc.
	Subject     string    `json:"subject" db:"subject"`
	Description *string   `json:"description" db:"description"`
	Outcome     *string   `json:"outcome" db:"outcome"`
	Priority    string    `json:"priority" db:"priority"` // low, medium, high, urgent
	Status      string    `json:"status" db:"status"`     // pending, completed, cancelled
	Duration    *int      `json:"duration" db:"duration"` // in minutes
	FollowUp    *time.Time `json:"follow_up" db:"follow_up"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (CustomerInteraction) TableName() string {
	return "customer_interactions"
}

func (CustomerInteraction) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS customer_interactions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		type TEXT NOT NULL CHECK (type IN ('call', 'email', 'meeting', 'support', 'sale', 'follow_up', 'other')),
		subject TEXT NOT NULL,
		description TEXT,
		outcome TEXT,
		priority TEXT DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
		status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
		duration INTEGER, -- in minutes
		follow_up TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);
	
	CREATE INDEX IF NOT EXISTS idx_customer_interactions_customer_id ON customer_interactions(customer_id);
	CREATE INDEX IF NOT EXISTS idx_customer_interactions_user_id ON customer_interactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_customer_interactions_type ON customer_interactions(type);
	CREATE INDEX IF NOT EXISTS idx_customer_interactions_status ON customer_interactions(status);
	CREATE INDEX IF NOT EXISTS idx_customer_interactions_created_at ON customer_interactions(created_at);
	`
}
