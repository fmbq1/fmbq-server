package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Slug         string     `json:"slug" db:"slug"`
	ParentID     *uuid.UUID `json:"parent_id" db:"parent_id"`
	Level        int        `json:"level" db:"level"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	ExternalCode *string    `json:"external_code" db:"external_code"`
	Metadata     string     `json:"metadata" db:"metadata"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

func (Category) TableName() string {
	return "categories"
}

func (Category) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS categories (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
		level INTEGER DEFAULT 1,
		is_active BOOLEAN DEFAULT true,
		external_code TEXT,
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);
	CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);
	CREATE INDEX IF NOT EXISTS idx_categories_level ON categories(level);`
}
