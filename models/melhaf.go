package models

import (
	"time"

	"github.com/google/uuid"
)

// MelhafType represents types like PERSI, Diana, etc.
type MelhafType struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	NameAr      *string   `json:"name_ar" db:"name_ar"`
	Description *string   `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (MelhafType) TableName() string {
	return "melhaf_types"
}

func (MelhafType) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_types (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		name_ar TEXT,
		description TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MelhafCollection represents a collection (each has multiple colors)
type MelhafCollection struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TypeID      uuid.UUID `json:"type_id" db:"type_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	SortOrder   int       `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (MelhafCollection) TableName() string {
	return "melhaf_collections"
}

func (MelhafCollection) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_collections (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		type_id UUID NOT NULL REFERENCES melhaf_types(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		description TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MelhafColor represents a color variant of a collection
type MelhafColor struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	CollectionID uuid.UUID  `json:"collection_id" db:"collection_id"`
	Name         string     `json:"name" db:"name"`
	NameAr       *string    `json:"name_ar" db:"name_ar"`
	ColorCode    *string    `json:"color_code" db:"color_code"` // Hex color code
	Price        float64    `json:"price" db:"price"`
	Discount     *float64   `json:"discount" db:"discount"` // Optional discount percentage
	EAN          *string    `json:"ean" db:"ean"`           // EAN code for barcode
	IsActive     bool       `json:"is_active" db:"is_active"`
	SortOrder    int        `json:"sort_order" db:"sort_order"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

func (MelhafColor) TableName() string {
	return "melhaf_colors"
}

func (MelhafColor) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_colors (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		collection_id UUID NOT NULL REFERENCES melhaf_collections(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		name_ar TEXT,
		color_code TEXT,
		price NUMERIC(12,2) NOT NULL DEFAULT 0,
		discount NUMERIC(5,2),
		ean TEXT,
		is_active BOOLEAN DEFAULT TRUE,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MelhafColorImage represents images for a color
type MelhafColorImage struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	ColorID   uuid.UUID  `json:"color_id" db:"color_id"`
	URL       string     `json:"url" db:"url"` // Cloudinary URL
	Alt       *string    `json:"alt" db:"alt"`
	Position  int        `json:"position" db:"position"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

func (MelhafColorImage) TableName() string {
	return "melhaf_color_images"
}

func (MelhafColorImage) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_color_images (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		color_id UUID NOT NULL REFERENCES melhaf_colors(id) ON DELETE CASCADE,
		url TEXT NOT NULL,
		alt TEXT,
		position INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MelhafVideo represents videos linked to collections (stored in Cloudinary)
type MelhafVideo struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	CollectionID uuid.UUID  `json:"collection_id" db:"collection_id"`
	Title        string     `json:"title" db:"title"`
	Description  *string    `json:"description" db:"description"`
	VideoURL     string     `json:"video_url" db:"video_url"` // Cloudinary URL
	ThumbnailURL *string    `json:"thumbnail_url" db:"thumbnail_url"` // Cloudinary thumbnail
	Duration     *int       `json:"duration" db:"duration"` // Duration in seconds
	IsActive     bool       `json:"is_active" db:"is_active"`
	SortOrder    int        `json:"sort_order" db:"sort_order"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

func (MelhafVideo) TableName() string {
	return "melhaf_videos"
}

func (MelhafVideo) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_videos (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		collection_id UUID NOT NULL REFERENCES melhaf_collections(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		description TEXT,
		video_url TEXT NOT NULL,
		thumbnail_url TEXT,
		duration INTEGER,
		is_active BOOLEAN DEFAULT TRUE,
		sort_order INTEGER DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MelhafInventory represents inventory/SKU for each color
type MelhafInventory struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ColorID      uuid.UUID `json:"color_id" db:"color_id"`
	Available    int       `json:"available" db:"available"`
	Reserved     int       `json:"reserved" db:"reserved"`
	ReorderPoint int       `json:"reorder_point" db:"reorder_point"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

func (MelhafInventory) TableName() string {
	return "melhaf_inventory"
}

func (MelhafInventory) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_inventory (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		color_id UUID NOT NULL UNIQUE REFERENCES melhaf_colors(id) ON DELETE CASCADE,
		available INTEGER NOT NULL DEFAULT 0,
		reserved INTEGER NOT NULL DEFAULT 0,
		reorder_point INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

// MelhafVideoLike represents a user's like on a video
type MelhafVideoLike struct {
	ID        uuid.UUID `json:"id" db:"id"`
	VideoID   uuid.UUID `json:"video_id" db:"video_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (MelhafVideoLike) TableName() string {
	return "melhaf_video_likes"
}

func (MelhafVideoLike) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_video_likes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		video_id UUID NOT NULL REFERENCES melhaf_videos(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		UNIQUE(video_id, user_id)
	);
	CREATE INDEX IF NOT EXISTS idx_melhaf_video_likes_video_id ON melhaf_video_likes (video_id);
	CREATE INDEX IF NOT EXISTS idx_melhaf_video_likes_user_id ON melhaf_video_likes (user_id);
	`
}

// MelhafVideoReaction represents a user's reaction (emoji) on a video
type MelhafVideoReaction struct {
	ID        uuid.UUID `json:"id" db:"id"`
	VideoID   uuid.UUID `json:"video_id" db:"video_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Reaction  string    `json:"reaction" db:"reaction"` // Emoji like "❤️", "🔥", "👍", etc.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (MelhafVideoReaction) TableName() string {
	return "melhaf_video_reactions"
}

func (MelhafVideoReaction) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS melhaf_video_reactions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		video_id UUID NOT NULL REFERENCES melhaf_videos(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		reaction TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		UNIQUE(video_id, user_id, reaction)
	);
	CREATE INDEX IF NOT EXISTS idx_melhaf_video_reactions_video_id ON melhaf_video_reactions (video_id);
	CREATE INDEX IF NOT EXISTS idx_melhaf_video_reactions_user_id ON melhaf_video_reactions (user_id);
	`
}

