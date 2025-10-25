package models

import (
	"time"

	"github.com/google/uuid"
)

type Price struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	SKUID     uuid.UUID  `json:"sku_id" db:"sku_id"`
	Currency  string     `json:"currency" db:"currency"`
	ListPrice float64    `json:"list_price" db:"list_price"`
	SalePrice *float64   `json:"sale_price" db:"sale_price"`
	StartAt   *time.Time `json:"start_at" db:"start_at"`
	EndAt     *time.Time `json:"end_at" db:"end_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

func (Price) TableName() string {
	return "prices"
}

func (Price) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS prices (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		sku_id UUID REFERENCES skus(id) ON DELETE CASCADE,
		currency CHAR(3) NOT NULL,
		list_price NUMERIC(12,2) NOT NULL,
		sale_price NUMERIC(12,2),
		start_at TIMESTAMP WITH TIME ZONE,
		end_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);
	CREATE INDEX IF NOT EXISTS idx_prices_sku_currency ON prices(sku_id, currency);`
}
