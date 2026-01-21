package models

import (
	"time"

	"github.com/google/uuid"
)

type ProductNotification struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	ProductID  uuid.UUID  `json:"product_id" db:"product_id"`
	SKUID      *uuid.UUID `json:"sku_id,omitempty" db:"sku_id"`
	NotifiedAt *time.Time `json:"notified_at,omitempty" db:"notified_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

func (ProductNotification) TableName() string {
	return "product_notifications"
}

func (ProductNotification) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS product_notifications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		product_id UUID NOT NULL REFERENCES product_models(id) ON DELETE CASCADE,
		sku_id UUID REFERENCES skus(id) ON DELETE CASCADE,
		notified_at TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_product_notifications_unique_with_sku 
		ON product_notifications(user_id, product_id, sku_id) 
		WHERE sku_id IS NOT NULL;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_product_notifications_unique_null_sku 
		ON product_notifications(user_id, product_id) 
		WHERE sku_id IS NULL;
	CREATE INDEX IF NOT EXISTS idx_product_notifications_product ON product_notifications(product_id);
	CREATE INDEX IF NOT EXISTS idx_product_notifications_user ON product_notifications(user_id);
	CREATE INDEX IF NOT EXISTS idx_product_notifications_sku ON product_notifications(sku_id) WHERE sku_id IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_product_notifications_pending ON product_notifications(product_id, notified_at) WHERE notified_at IS NULL;`
}
