package models

import (
	"time"

	"github.com/google/uuid"
)

type ProductImage struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ProductModelID uuid.UUID  `json:"product_model_id" db:"product_model_id"`
	ProductColorID uuid.UUID  `json:"product_color_id" db:"product_color_id"`
	SKUID          *uuid.UUID `json:"sku_id" db:"sku_id"`
	URL            string     `json:"url" db:"url"`
	Alt            *string    `json:"alt" db:"alt"`
	Position       int        `json:"position" db:"position"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

func (ProductImage) TableName() string {
	return "product_images"
}

func (ProductImage) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS product_images (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_model_id UUID REFERENCES product_models(id) ON DELETE CASCADE,
		product_color_id UUID REFERENCES product_colors(id) ON DELETE CASCADE,
		sku_id UUID REFERENCES skus(id) ON DELETE SET NULL,
		url TEXT NOT NULL,
		alt TEXT,
		position INT DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
