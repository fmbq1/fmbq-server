package models

import (
	"time"

	"github.com/google/uuid"
)

// AddressBook represents a user's address book entry
type AddressBook struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Label       string     `json:"label" db:"label"` // Home, Work, etc.
	City        string     `json:"city" db:"city"`
	Quartier    string     `json:"quartier" db:"quartier"`
	Street      *string    `json:"street,omitempty" db:"street"`
	Building    *string    `json:"building,omitempty" db:"building"`
	Floor       *string    `json:"floor,omitempty" db:"floor"`
	Apartment    *string    `json:"apartment,omitempty" db:"apartment"`
	Latitude    *float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude   *float64   `json:"longitude,omitempty" db:"longitude"`
	IsDefault   bool       `json:"is_default" db:"is_default"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// City represents a Mauritanian city
type City struct {
	ID       uuid.UUID `json:"id" db:"id"`
	Name     string    `json:"name" db:"name"`
	NameAr   string    `json:"name_ar" db:"name_ar"` // Arabic name
	Region   string    `json:"region" db:"region"`
	IsActive bool      `json:"is_active" db:"is_active"`
}

// Quartier represents a neighborhood/quartier in a city
type Quartier struct {
	ID       uuid.UUID `json:"id" db:"id"`
	CityID   uuid.UUID `json:"city_id" db:"city_id"`
	Name     string    `json:"name" db:"name"`
	NameAr   string    `json:"name_ar" db:"name_ar"` // Arabic name
	IsActive bool      `json:"is_active" db:"is_active"`
}

func (AddressBook) TableName() string {
	return "address_book"
}

func (City) TableName() string {
	return "cities"
}

func (Quartier) TableName() string {
	return "quartiers"
}

func (AddressBook) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS address_book (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		label TEXT NOT NULL,
		city TEXT NOT NULL,
		quartier TEXT NOT NULL,
		street TEXT,
		building TEXT,
		floor TEXT,
		apartment TEXT,
		latitude DOUBLE PRECISION,
		longitude DOUBLE PRECISION,
		is_default BOOLEAN DEFAULT FALSE,
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`
}

func (City) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS cities (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		name_ar TEXT,
		region TEXT NOT NULL,
		is_active BOOLEAN DEFAULT TRUE
	);`
}

func (Quartier) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS quartiers (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		city_id UUID NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		name_ar TEXT,
		is_active BOOLEAN DEFAULT TRUE
	);`
}
