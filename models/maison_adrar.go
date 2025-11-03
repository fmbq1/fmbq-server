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
	// Extended fields to match high-quality schema
	GenderCategory   *string `json:"gender_category" db:"gender_category"`       // Men/Women/Unisex
	Concentration    *string `json:"concentration" db:"concentration"`           // EDP/EDT/Parfum
	FragranceFamily  *string `json:"fragrance_family" db:"fragrance_family"`     // Woody/Floral
	TopNotes         *string `json:"top_notes" db:"top_notes"`                   // JSON string or text
	MiddleNotes      *string `json:"middle_notes" db:"middle_notes"`
	BaseNotes        *string `json:"base_notes" db:"base_notes"`
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
		gender_category TEXT,
		concentration TEXT,
		fragrance_family TEXT,
		top_notes TEXT,
		middle_notes TEXT,
		base_notes TEXT,
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
	// Treat as variant
	Name       string    `json:"name" db:"name"`              // color_name / edition name
	NameAr     *string   `json:"name_ar" db:"name_ar"`
	ColorCode  *string   `json:"color_code" db:"color_code"`  // hex_color
	Price      float64   `json:"price" db:"price"`            // base price / legacy
	PriceOverride *float64 `json:"price_override" db:"price_override"`
	VolumeML   *int      `json:"volume_ml" db:"volume_ml"`
	Stock      *int      `json:"stock" db:"stock"`
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
		price_override NUMERIC(12,2),
		volume_ml INTEGER,
		stock INTEGER,
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
	IsMain     bool      `json:"is_main" db:"is_main"`
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
		is_main BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// Feed blocks (Zalando/FashionNova style)
type FeedBlock struct {
    ID        uuid.UUID `json:"id" db:"id"`
    Type      string    `json:"type" db:"type"` // banner, carousel, grid, story, video_section
    Title     *string   `json:"title" db:"title"`
    Subtitle  *string   `json:"subtitle" db:"subtitle"`
    BackgroundImageURL *string `json:"background_image_url" db:"background_image_url"`
    CTALabel  *string   `json:"cta_label" db:"cta_label"`
    CTAAction *string   `json:"cta_action" db:"cta_action"` // JSON
    Position  int       `json:"position" db:"position"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (FeedBlock) TableName() string { return "feed_blocks" }
func (FeedBlock) CreateTableSQL() string {
    return `
    CREATE TABLE IF NOT EXISTS feed_blocks (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        type VARCHAR(50) NOT NULL,
        title VARCHAR(255),
        subtitle VARCHAR(255),
        background_image_url TEXT,
        cta_label VARCHAR(100),
        cta_action JSONB,
        position INTEGER DEFAULT 0,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
    );`
}

type FeedBlockItem struct {
    ID          uuid.UUID `json:"id" db:"id"`
    FeedBlockID uuid.UUID `json:"feed_block_id" db:"feed_block_id"`
    PerfumeID   uuid.UUID `json:"perfume_id" db:"perfume_id"`
    CustomImageURL *string `json:"custom_image_url" db:"custom_image_url"`
    HighlightText *string `json:"highlight_text" db:"highlight_text"`
    Position    int       `json:"position" db:"position"`
}

func (FeedBlockItem) TableName() string { return "feed_block_items" }
func (FeedBlockItem) CreateTableSQL() string {
    return `
    CREATE TABLE IF NOT EXISTS feed_block_items (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        feed_block_id UUID NOT NULL REFERENCES feed_blocks(id) ON DELETE CASCADE,
        perfume_id UUID NOT NULL REFERENCES maison_adrar_perfumes(id) ON DELETE CASCADE,
        custom_image_url TEXT,
        highlight_text VARCHAR(255),
        position INTEGER DEFAULT 0
    );`
}

type Campaign struct {
    ID           uuid.UUID `json:"id" db:"id"`
    Name         string    `json:"name" db:"name"`
    Tagline      *string   `json:"tagline" db:"tagline"`
    StartDate    *time.Time `json:"start_date" db:"start_date"`
    EndDate      *time.Time `json:"end_date" db:"end_date"`
    HeroImageURL *string   `json:"hero_image_url" db:"hero_image_url"`
}

func (Campaign) TableName() string { return "campaigns" }
func (Campaign) CreateTableSQL() string { return `
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    tagline TEXT,
    start_date DATE,
    end_date DATE,
    hero_image_url TEXT
);` }

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

