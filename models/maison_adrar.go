package models

import (
	"time"

	"github.com/google/uuid"
)

// MaisonAdrarCategory represents perfume categories (House Perfume, Eau de Parfum, etc.)
type MaisonAdrarCategory struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	NameAr      *string   `json:"name_ar" db:"name_ar"`
	Description *string   `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (MaisonAdrarCategory) TableName() string {
	return "maison_adrar_categories"
}

func (MaisonAdrarCategory) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS maison_adrar_categories (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		name_ar TEXT,
		description TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MaisonAdrarCollection represents a perfume collection (grouped by background)
type MaisonAdrarCollection struct {
	ID             uuid.UUID `json:"id" db:"id"`
	CategoryID     *uuid.UUID `json:"category_id" db:"category_id"` // Optional - some perfumes have no category
	Name           string    `json:"name" db:"name"`                 // Section title
	Description    *string   `json:"description" db:"description"`
	BackgroundColor *string   `json:"background_color" db:"background_color"` // Background color (hex)
	BackgroundURL  *string   `json:"background_url" db:"background_url"`      // Background image for grouping
	BannerURL      *string   `json:"banner_url" db:"banner_url"`             // Section banner image
	IsActive       bool      `json:"is_active" db:"is_active"`
	SortOrder      int       `json:"sort_order" db:"sort_order"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

func (MaisonAdrarCollection) TableName() string {
	return "maison_adrar_collections"
}

func (MaisonAdrarCollection) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS maison_adrar_collections (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		category_id UUID REFERENCES maison_adrar_categories(id) ON DELETE SET NULL,
		name TEXT NOT NULL,
		description TEXT,
		background_color TEXT,
		background_url TEXT,
		banner_url TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MaisonAdrarPerfume represents a perfume (can have multiple colors/variants)
type MaisonAdrarPerfume struct {
	ID           uuid.UUID `json:"id" db:"id"`
	CollectionID uuid.UUID `json:"collection_id" db:"collection_id"`
	Name         string    `json:"name" db:"name"`
	NameAr       *string   `json:"name_ar" db:"name_ar"`
	Type         *string   `json:"type" db:"type"`                    // Perfume type (Eau de Parfum, Eau de Toilette, etc.)
	Size         *string   `json:"size" db:"size"`                    // Size (100ML, 50ML, etc.)
	Description  *string   `json:"description" db:"description"`
	Ingredients  *string   `json:"ingredients" db:"ingredients"`      // Ingredients list
	Price        float64   `json:"price" db:"price"`
	Discount     *float64  `json:"discount" db:"discount"`            // Optional discount percentage
	EAN          *string   `json:"ean" db:"ean"`                      // EAN code for barcode
	IsActive     bool      `json:"is_active" db:"is_active"`
	SortOrder    int       `json:"sort_order" db:"sort_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

func (MaisonAdrarPerfume) TableName() string {
	return "maison_adrar_perfumes"
}

func (MaisonAdrarPerfume) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS maison_adrar_perfumes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		collection_id UUID NOT NULL REFERENCES maison_adrar_collections(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		name_ar TEXT,
		type TEXT,
		size TEXT,
		description TEXT,
		ingredients TEXT,
		price NUMERIC(12,2) NOT NULL DEFAULT 0,
		discount NUMERIC(5,2),
		ean TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MaisonAdrarPerfumeColor represents a color/variant of a perfume
type MaisonAdrarPerfumeColor struct {
	ID         uuid.UUID `json:"id" db:"id"`
	PerfumeID  uuid.UUID `json:"perfume_id" db:"perfume_id"`
	Name       string    `json:"name" db:"name"`
	NameAr     *string   `json:"name_ar" db:"name_ar"`
	ColorCode  *string   `json:"color_code" db:"color_code"` // Hex color code
	Price      float64   `json:"price" db:"price"`            // Override perfume price if different
	Discount   *float64  `json:"discount" db:"discount"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	SortOrder  int       `json:"sort_order" db:"sort_order"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

func (MaisonAdrarPerfumeColor) TableName() string {
	return "maison_adrar_perfume_colors"
}

func (MaisonAdrarPerfumeColor) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS maison_adrar_perfume_colors (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		perfume_id UUID NOT NULL REFERENCES maison_adrar_perfumes(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		name_ar TEXT,
		color_code TEXT,
		price NUMERIC(12,2) NOT NULL DEFAULT 0,
		discount NUMERIC(5,2),
		is_active BOOLEAN DEFAULT TRUE,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MaisonAdrarPerfumeImage represents images directly linked to a perfume
type MaisonAdrarPerfumeImage struct {
	ID         uuid.UUID `json:"id" db:"id"`
	PerfumeID  uuid.UUID `json:"perfume_id" db:"perfume_id"`
	URL        string    `json:"url" db:"url"`
	Alt        *string   `json:"alt" db:"alt"`
	Position   int       `json:"position" db:"position"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

func (MaisonAdrarPerfumeImage) TableName() string {
	return "maison_adrar_perfume_images"
}

func (MaisonAdrarPerfumeImage) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS maison_adrar_perfume_images (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		perfume_id UUID NOT NULL REFERENCES maison_adrar_perfumes(id) ON DELETE CASCADE,
		url TEXT NOT NULL,
		alt TEXT,
		position INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MaisonAdrarBanner represents banners for the perfume feed
type MaisonAdrarBanner struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CategoryID  *uuid.UUID `json:"category_id" db:"category_id"` // Optional
	Title       string    `json:"title" db:"title"`
	ImageURL    string    `json:"image_url" db:"image_url"`
	LinkURL     *string   `json:"link_url" db:"link_url"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (MaisonAdrarBanner) TableName() string {
	return "maison_adrar_banners"
}

func (MaisonAdrarBanner) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS maison_adrar_banners (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		category_id UUID REFERENCES maison_adrar_categories(id) ON DELETE SET NULL,
		title TEXT NOT NULL,
		image_url TEXT NOT NULL,
		link_url TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MaisonAdrarInventory represents inventory for perfume colors
type MaisonAdrarInventory struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ColorID      uuid.UUID `json:"color_id" db:"color_id"`
	Available    int       `json:"available" db:"available"`
	Reserved     int       `json:"reserved" db:"reserved"`
	ReorderPoint int       `json:"reorder_point" db:"reorder_point"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

func (MaisonAdrarInventory) TableName() string {
	return "maison_adrar_inventory"
}

func (MaisonAdrarInventory) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS maison_adrar_inventory (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		color_id UUID NOT NULL REFERENCES maison_adrar_perfume_colors(id) ON DELETE CASCADE,
		available INTEGER DEFAULT 0,
		reserved INTEGER DEFAULT 0,
		reorder_point INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

