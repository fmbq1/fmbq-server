package models

import (
	"time"

	"github.com/google/uuid"
)

type Inventory struct {
	SKUID      uuid.UUID `json:"sku_id" db:"sku_id"`
	Available  int       `json:"available" db:"available"`
	Reserved   int       `json:"reserved" db:"reserved"`
    ReorderPoint int     `json:"reorder_point" db:"reorder_point"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

func (Inventory) TableName() string {
	return "inventory"
}

func (Inventory) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS inventory (
		sku_id UUID PRIMARY KEY REFERENCES skus(id) ON DELETE CASCADE,
		available INT NOT NULL DEFAULT 0,
		reserved INT NOT NULL DEFAULT 0,
        reorder_point INT NOT NULL DEFAULT 0,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
