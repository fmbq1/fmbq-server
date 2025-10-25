package models

import (
	"time"

	"github.com/google/uuid"
)

type CartItem struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CartID    uuid.UUID `json:"cart_id" db:"cart_id"`
	SKUID     uuid.UUID `json:"sku_id" db:"sku_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
	AddedAt   time.Time `json:"added_at" db:"added_at"`
}

func (CartItem) TableName() string {
	return "cart_items"
}

func (CartItem) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS cart_items (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		cart_id UUID REFERENCES carts(id) ON DELETE CASCADE,
		sku_id UUID REFERENCES skus(id),
		quantity INT NOT NULL DEFAULT 1,
		added_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
