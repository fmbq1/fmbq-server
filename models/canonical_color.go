package models

import (
	"time"

	"github.com/google/uuid"
)

// CanonicalColor represents a normalized color that can be mapped to various color names
// This prevents duplicate colors and language mismatches
type CanonicalColor struct {
	ID         uuid.UUID `json:"id" db:"id"`
	InternalKey string   `json:"internal_key" db:"internal_key"` // e.g., "white", "gold", "blue"
	NameEn     string    `json:"name_en" db:"name_en"`           // English display name
	NameFr     string    `json:"name_fr" db:"name_fr"`           // French display name
	ColorCode  *string   `json:"color_code" db:"color_code"`     // Hex color code (optional)
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

func (CanonicalColor) TableName() string {
	return "canonical_colors"
}

func (CanonicalColor) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS canonical_colors (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		internal_key TEXT NOT NULL UNIQUE,
		name_en TEXT NOT NULL,
		name_fr TEXT NOT NULL,
		color_code TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);
	CREATE INDEX IF NOT EXISTS idx_canonical_colors_internal_key ON canonical_colors(internal_key);
	CREATE INDEX IF NOT EXISTS idx_canonical_colors_active ON canonical_colors(is_active);`
}

// MelhafColorCanonicalMatch links Melhafa colors to canonical colors
// This allows a Melhafa color to be matched to products via canonical colors
type MelhafColorCanonicalMatch struct {
	ID              uuid.UUID `json:"id" db:"id"`
	MelhafColorID   uuid.UUID `json:"melhaf_color_id" db:"melhaf_color_id"`
	CanonicalColorID uuid.UUID `json:"canonical_color_id" db:"canonical_color_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

func (MelhafColorCanonicalMatch) TableName() string {
	return "melhaf_color_canonical_matches"
}

func (MelhafColorCanonicalMatch) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_color_canonical_matches (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		melhaf_color_id UUID NOT NULL REFERENCES melhaf_colors(id) ON DELETE CASCADE,
		canonical_color_id UUID NOT NULL REFERENCES canonical_colors(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		UNIQUE(melhaf_color_id, canonical_color_id)
	);
	CREATE INDEX IF NOT EXISTS idx_melhaf_color_matches_color ON melhaf_color_canonical_matches(melhaf_color_id);
	CREATE INDEX IF NOT EXISTS idx_melhaf_color_matches_canonical ON melhaf_color_canonical_matches(canonical_color_id);`
}

// ProductColorCanonicalMatch links product colors to canonical colors
// This allows products to be matched to Melhafas via canonical colors
type ProductColorCanonicalMatch struct {
	ID              uuid.UUID `json:"id" db:"id"`
	ProductColorID  uuid.UUID `json:"product_color_id" db:"product_color_id"`
	CanonicalColorID uuid.UUID `json:"canonical_color_id" db:"canonical_color_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

func (ProductColorCanonicalMatch) TableName() string {
	return "product_color_canonical_matches"
}

func (ProductColorCanonicalMatch) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS product_color_canonical_matches (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_color_id UUID NOT NULL REFERENCES product_colors(id) ON DELETE CASCADE,
		canonical_color_id UUID NOT NULL REFERENCES canonical_colors(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		UNIQUE(product_color_id, canonical_color_id)
	);
	CREATE INDEX IF NOT EXISTS idx_product_color_matches_color ON product_color_canonical_matches(product_color_id);
	CREATE INDEX IF NOT EXISTS idx_product_color_matches_canonical ON product_color_canonical_matches(canonical_color_id);`
}
