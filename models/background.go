package models

import (
	"time"
)

type Background struct {
	ID          string    `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	ImageURL    string    `json:"image_url" db:"image_url"`
	Position    int       `json:"position" db:"position"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	CategoryID  *string   `json:"category_id" db:"category_id"`
	ActionType  string    `json:"action_type" db:"action_type"` // search, category, brand, product, url
	ActionData  string    `json:"action_data" db:"action_data"` // JSON string with filter data
	StartDate   *time.Time `json:"start_date" db:"start_date"`
	EndDate     *time.Time `json:"end_date" db:"end_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (Background) TableName() string { return "backgrounds" }

func (Background) CreateTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS backgrounds (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        title VARCHAR(255) NOT NULL,
        description TEXT,
        image_url TEXT NOT NULL,
        position INTEGER DEFAULT 0,
        is_active BOOLEAN DEFAULT true,
        category_id UUID REFERENCES categories(id),
        action_type VARCHAR(50) DEFAULT 'search',
        action_data JSONB,
        start_date TIMESTAMP,
        end_date TIMESTAMP,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
    );`
}
