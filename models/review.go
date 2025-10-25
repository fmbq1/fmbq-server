package models

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID             uuid.UUID `json:"id" db:"id"`
	ProductModelID uuid.UUID `json:"product_model_id" db:"product_model_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Rating         int       `json:"rating" db:"rating"`
	Title          *string   `json:"title" db:"title"`
	Body           *string   `json:"body" db:"body"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

func (Review) TableName() string {
	return "reviews"
}

func (Review) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS reviews (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_model_id UUID REFERENCES product_models(id) ON DELETE CASCADE,
		user_id UUID REFERENCES users(id),
		rating SMALLINT CHECK (rating >= 1 AND rating <= 5),
		title TEXT,
		body TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
