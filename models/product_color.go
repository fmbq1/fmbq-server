package models

import (
	"time"

	"github.com/google/uuid"
)

type ProductColor struct {
	ID                uuid.UUID `json:"id" db:"id"`
	ProductModelID    uuid.UUID `json:"product_model_id" db:"product_model_id"`
	ColorName         string    `json:"color_name" db:"color_name"`
	ColorCode         *string   `json:"color_code" db:"color_code"`
	ExternalColorID   *string   `json:"external_color_id" db:"external_color_id"`
	DefaultImageID    *uuid.UUID `json:"default_image_id" db:"default_image_id"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

func (ProductColor) TableName() string {
	return "product_colors"
}

func (ProductColor) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS product_colors (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_model_id UUID REFERENCES product_models(id) ON DELETE CASCADE,
		color_name TEXT NOT NULL,
		color_code TEXT,
		external_color_id TEXT,
		default_image_id UUID,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
