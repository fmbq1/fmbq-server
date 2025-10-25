package models

import (
	"time"

	"github.com/google/uuid"
)

type LoyaltyTransaction struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Delta       int64      `json:"delta" db:"delta"`
	Reason      *string    `json:"reason" db:"reason"`
	ReferenceID *uuid.UUID `json:"reference_id" db:"reference_id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

func (LoyaltyTransaction) TableName() string {
	return "loyalty_transactions"
}

func (LoyaltyTransaction) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS loyalty_transactions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		delta BIGINT NOT NULL,
		reason TEXT,
		reference_id UUID,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
