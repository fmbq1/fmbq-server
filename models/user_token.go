package models

import (
	"time"

	"github.com/google/uuid"
)

type UserToken struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	LastUsed  *time.Time `json:"last_used" db:"last_used"`
	Revoked   bool      `json:"revoked" db:"revoked"`
}

func (UserToken) TableName() string {
	return "user_tokens"
}

func (UserToken) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS user_tokens (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token TEXT UNIQUE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
		last_used TIMESTAMP WITH TIME ZONE,
		revoked BOOLEAN DEFAULT FALSE,
		CONSTRAINT user_tokens_token_key UNIQUE (token)
	);
	
	CREATE INDEX IF NOT EXISTS idx_user_tokens_user_id ON user_tokens(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_tokens_token ON user_tokens(token);
	CREATE INDEX IF NOT EXISTS idx_user_tokens_revoked ON user_tokens(revoked);
	`
}

