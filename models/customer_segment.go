package models

import (
	"time"

	"github.com/google/uuid"
)

type CustomerSegment struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description" db:"description"`
	Criteria    string    `json:"criteria" db:"criteria"` // JSON criteria for segmentation
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CustomerSegmentMember struct {
	ID         uuid.UUID `json:"id" db:"id"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id"`
	SegmentID  uuid.UUID `json:"segment_id" db:"segment_id"`
	AddedAt    time.Time `json:"added_at" db:"added_at"`
}

func (CustomerSegment) TableName() string {
	return "customer_segments"
}

func (CustomerSegmentMember) TableName() string {
	return "customer_segment_members"
}

func (CustomerSegment) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS customer_segments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		criteria JSONB NOT NULL,
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
	);
	
	CREATE TABLE IF NOT EXISTS customer_segment_members (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
		segment_id UUID NOT NULL REFERENCES customer_segments(id) ON DELETE CASCADE,
		added_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
		UNIQUE(customer_id, segment_id)
	);
	
	CREATE INDEX IF NOT EXISTS idx_customer_segment_members_customer_id ON customer_segment_members(customer_id);
	CREATE INDEX IF NOT EXISTS idx_customer_segment_members_segment_id ON customer_segment_members(segment_id);
	`
}
