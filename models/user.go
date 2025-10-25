package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        *string   `json:"email" db:"email"`
	Phone        *string   `json:"phone" db:"phone"`
	PasswordHash *string   `json:"password_hash" db:"password_hash"`
	FullName     *string   `json:"full_name" db:"full_name"`
	Avatar       *string   `json:"avatar,omitempty" db:"avatar"`
	Role        string    `json:"role" db:"role"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	Metadata     string    `json:"metadata" db:"metadata"`
}

func (User) TableName() string {
	return "users"
}

func (User) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email TEXT UNIQUE,
		phone TEXT UNIQUE,
		password_hash TEXT,
		full_name TEXT,
		avatar TEXT,
		role TEXT DEFAULT 'user',
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		metadata JSONB DEFAULT '{}'
	);`
}
