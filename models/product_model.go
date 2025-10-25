package models

import (
	"time"

	"github.com/google/uuid"
)

type ProductModel struct {
	ID               uuid.UUID `json:"id" db:"id"`
	BrandID          uuid.UUID `json:"brand_id" db:"brand_id"`
	Title            string    `json:"title" db:"title"`
	Description      *string   `json:"description" db:"description"`
	ShortDescription *string   `json:"short_description" db:"short_description"`
	ModelCode        *string   `json:"model_code" db:"model_code"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	Attributes       string    `json:"attributes" db:"attributes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

func (ProductModel) TableName() string {
	return "product_models"
}

func (ProductModel) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS product_models (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		brand_id UUID REFERENCES brands(id),
		title TEXT NOT NULL,
		description TEXT,
		short_description TEXT,
		model_code TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		attributes JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
