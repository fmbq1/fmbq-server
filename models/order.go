package models

import (
	"time"

	"github.com/google/uuid"
)

// Order represents an order in the system
type Order struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	UserID           uuid.UUID      `json:"user_id" db:"user_id"`
	OrderNumber      string         `json:"order_number" db:"order_number"`
	Status           string         `json:"status" db:"status"`
	TotalAmount      float64        `json:"total_amount" db:"total_amount"`
	DeliveryOption   string         `json:"delivery_option" db:"delivery_option"`
	DeliveryAddress  DeliveryAddress `json:"delivery_address" db:"delivery_address"`
	PaymentProof     string         `json:"payment_proof" db:"payment_proof"`
	PromotionalCode  *string        `json:"promotional_code,omitempty" db:"promotional_code"`
	DiscountAmount   float64        `json:"discount_amount" db:"discount_amount"`
	DeliveryZoneQuartierID   *string `json:"delivery_zone_quartier_id,omitempty" db:"delivery_zone_quartier_id"`
	DeliveryZoneQuartierName *string `json:"delivery_zone_quartier_name,omitempty" db:"delivery_zone_quartier_name"`
	DeliveryZoneFee          float64 `json:"delivery_zone_fee" db:"delivery_zone_fee"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
	Items            []OrderItem    `json:"items,omitempty"`
}

// OrderItem represents an item within an order
type OrderItem struct {
	ID           uuid.UUID `json:"id" db:"id"`
	OrderID      uuid.UUID `json:"order_id" db:"order_id"`
	ProductID    uuid.UUID `json:"product_id" db:"product_id"`
	SKUID        uuid.UUID `json:"sku_id" db:"sku_id"`
	Quantity     int       `json:"quantity" db:"quantity"`
	UnitPrice    float64   `json:"unit_price" db:"unit_price"`
	Size         *string   `json:"size" db:"size"`
	Color        *string   `json:"color" db:"color"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	// Additional fields for display
	ProductName  string    `json:"product_name,omitempty"`
	BrandName    string    `json:"brand_name,omitempty"`
	ProductImage string    `json:"product_image,omitempty"`
}

// DeliveryAddress represents a delivery address for orders
type DeliveryAddress struct {
	AddressID  string   `json:"address_id"`
	City       string   `json:"city"`
	Quartier   string   `json:"quartier"`
	Street     *string  `json:"street,omitempty"`
	Building   *string  `json:"building,omitempty"`
	Floor      *string  `json:"floor,omitempty"`
	Apartment  *string  `json:"apartment,omitempty"`
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
}

// TableName specifies the table name for GORM
func (Order) TableName() string {
	return "orders"
}

// TableName specifies the table name for GORM
func (OrderItem) TableName() string {
	return "order_items"
}

// CreateTableSQL returns the SQL for creating the orders table
func (Order) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS orders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		order_number VARCHAR(50) NOT NULL UNIQUE,
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		total_amount NUMERIC(12,2) NOT NULL,
		delivery_option VARCHAR(20) NOT NULL CHECK (delivery_option IN ('pickup', 'delivery')),
		delivery_address JSONB NOT NULL,
		payment_proof TEXT NOT NULL,
		promotional_code VARCHAR(50),
		discount_amount NUMERIC(12,2) DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`
}

// CreateTableSQL returns the SQL for creating the order_items table
func (OrderItem) CreateTableSQL() string {
	return `
	CREATE TABLE IF NOT EXISTS order_items (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		product_id UUID NOT NULL REFERENCES product_models(id) ON DELETE CASCADE,
		sku_id UUID NOT NULL REFERENCES skus(id) ON DELETE CASCADE,
		quantity INTEGER NOT NULL CHECK (quantity > 0),
		unit_price NUMERIC(12,2) NOT NULL,
		size VARCHAR(50),
		color VARCHAR(50),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`
}