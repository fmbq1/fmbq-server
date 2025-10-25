package models

import (
	"time"

	"github.com/google/uuid"
)

type LoyaltyAccount struct {
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	PointsBalance int64     `json:"points_balance" db:"points_balance"`
	Tier          string    `json:"tier" db:"tier"`
	TotalEarned   int64     `json:"total_earned" db:"total_earned"`
	TotalRedeemed int64     `json:"total_redeemed" db:"total_redeemed"`
	LastActivity  time.Time `json:"last_activity" db:"last_activity"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

func (LoyaltyAccount) TableName() string {
	return "loyalty_accounts"
}

func (LoyaltyAccount) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS loyalty_accounts (
		user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		points_balance BIGINT DEFAULT 0,
		tier TEXT DEFAULT 'bronze' CHECK (tier IN ('bronze', 'silver', 'gold', 'platinum')),
		total_earned BIGINT DEFAULT 0,
		total_redeemed BIGINT DEFAULT 0,
		last_activity TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
