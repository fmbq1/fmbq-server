package models

import "github.com/google/uuid"

type ProductModelCategory struct {
	ProductModelID uuid.UUID `json:"product_model_id" db:"product_model_id"`
	CategoryID     uuid.UUID `json:"category_id" db:"category_id"`
}

func (ProductModelCategory) TableName() string {
	return "product_model_categories"
}

func (ProductModelCategory) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS product_model_categories (
		product_model_id UUID REFERENCES product_models(id) ON DELETE CASCADE,
		category_id UUID REFERENCES categories(id) ON DELETE CASCADE,
		PRIMARY KEY (product_model_id, category_id)
	);`
}
