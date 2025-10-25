package models

import "time"

type Banner struct {
	ID        string    `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	ImageURL  string    `json:"image_url" db:"image_url"`
	Link      string    `json:"link" db:"link"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (Banner) TableName() string { return "banners" }

func (Banner) CreateTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS banners (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        title VARCHAR(255) NOT NULL,
        image_url TEXT NOT NULL,
        link TEXT,
        sort_order INTEGER DEFAULT 0,
        is_active BOOLEAN DEFAULT true,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
    );`
}
