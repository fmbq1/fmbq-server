package models

import (
	"time"

	"github.com/google/uuid"
)

type SKU struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	ProductModelID  uuid.UUID  `json:"product_model_id" db:"product_model_id"`
	ProductColorID  uuid.UUID  `json:"product_color_id" db:"product_color_id"`
	SKUCode         string     `json:"sku_code" db:"sku_code"`
	EAN             *string    `json:"ean" db:"ean"`
	Size            *string    `json:"size" db:"size"`
	SizeNormalized  *string    `json:"size_normalized" db:"size_normalized"`
	SizeChartID     *uuid.UUID `json:"size_chart_id" db:"size_chart_id"`
	Attributes      string     `json:"attributes" db:"attributes"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

func (SKU) TableName() string {
	return "skus"
}

func (SKU) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS skus (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_model_id UUID REFERENCES product_models(id) ON DELETE CASCADE,
		product_color_id UUID REFERENCES product_colors(id) ON DELETE CASCADE,
		sku_code TEXT NOT NULL UNIQUE,
		ean TEXT,
		size TEXT,
		size_normalized TEXT,
		size_chart_id UUID REFERENCES size_charts(id),
		attributes JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);
	CREATE INDEX IF NOT EXISTS idx_skus_product_color ON skus(product_color_id);
	CREATE INDEX IF NOT EXISTS idx_skus_model_size ON skus(product_model_id, size);`
}
