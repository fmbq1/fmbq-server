package models

import (
	"time"

	"github.com/google/uuid"
)

type SizeChart struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	ChartJSON string    `json:"chart_json" db:"chart_json"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (SizeChart) TableName() string {
	return "size_charts"
}

func (SizeChart) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS size_charts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		chart_json JSONB NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}
