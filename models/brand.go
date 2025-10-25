package models

import (
	"time"

	"github.com/google/uuid"
)

type Brand struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Slug         *string   `json:"slug" db:"slug"`
	Description  *string   `json:"description" db:"description"`
	ExternalCode *string   `json:"external_code" db:"external_code"`
	Logo         *string   `json:"logo" db:"logo"`
	Banner       *string   `json:"banner" db:"banner"`
	Color        *string   `json:"color" db:"color"`
    ParentCategoryID *uuid.UUID `json:"parent_category_id" db:"parent_category_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

func (Brand) TableName() string {
	return "brands"
}

func (Brand) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS brands (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		slug TEXT UNIQUE,
		description TEXT,
		external_code TEXT,
		logo TEXT,
		banner TEXT,
		color TEXT,
        parent_category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
