package database

import (
	"database/sql"
	"fmt"
	"log"

	"fmbq-server/models"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

var Database *DB

// Connect establishes a connection to the PostgreSQL database
func Connect(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	Database = &DB{db}
	return Database, nil
}

// InitializeTables creates all tables if they don't exist
func (db *DB) InitializeTables() error {
	// Enable pgcrypto extension
	if _, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`); err != nil {
		return fmt.Errorf("failed to enable pgcrypto extension: %w", err)
	}

	// Define the order of table creation (respecting foreign key dependencies)
	models := []interface{}{
		models.Category{},
		models.Brand{},
		models.ProductModel{},
		models.ProductModelCategory{},
		models.SizeChart{},
		models.ProductColor{},
		models.SKU{},
		models.ProductImage{},
		models.Inventory{},
		models.Price{},
		models.User{},
		models.LoyaltyAccount{},
		models.LoyaltyTransaction{},
		models.Address{},
		models.Order{},
		models.OrderItem{},
		models.Cart{},
		models.CartItem{},
		models.WishlistItem{},
		models.Review{},
		// CRM models
		models.Customer{},
		models.CustomerInteraction{},
		models.CustomerSegment{},
		models.CustomerSegmentMember{},
		// Payment models
		models.PaymentMethod{},
		models.Banner{},
		models.Background{},
		// Melhaf models
		models.MelhafType{},
		models.MelhafCollection{},
		models.MelhafColor{},
		models.MelhafColorImage{},
		models.MelhafVideo{},
		models.MelhafInventory{},
		// Maison Adrar models
		models.MaisonAdrarCategory{},
		models.MaisonAdrarCollection{},
		models.MaisonAdrarPerfume{},
		models.MaisonAdrarPerfumeImage{},
		models.MaisonAdrarBanner{},
		models.FeedBlock{},
		models.FeedBlockItem{},
		models.Campaign{},
	}

	for _, model := range models {
		if tableModel, ok := model.(interface {
			TableName() string
			CreateTableSQL() string
		}); ok {
			tableName := tableModel.TableName()
			createSQL := tableModel.CreateTableSQL()
			
			log.Printf("Creating table: %s", tableName)
			if _, err := db.Exec(createSQL); err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		}
	}

	// Run schema migrations
	if err := db.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("All tables created successfully!")
	return nil
}

// runMigrations handles schema updates for existing tables
func (db *DB) runMigrations() error {
	migrations := []string{
		// Add role column to users table if it doesn't exist
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user';`,
		
		// Add is_active column to users table if it doesn't exist
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;`,
		
		// Add avatar column to users table if it doesn't exist
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar TEXT;`,
		
		// Add email column to users table if it doesn't exist
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT UNIQUE;`,
		
		// Add timestamp columns to cities and quartiers if they don't exist
		`ALTER TABLE cities ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT now();`,
		`ALTER TABLE cities ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT now();`,
		`ALTER TABLE quartiers ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT now();`,
		`ALTER TABLE quartiers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT now();`,
		
		// Add background_color and banner_url columns to maison_adrar_collections if they don't exist
		`ALTER TABLE maison_adrar_collections ADD COLUMN IF NOT EXISTS background_color TEXT;`,
		`ALTER TABLE maison_adrar_collections ADD COLUMN IF NOT EXISTS banner_url TEXT;`,

    // Ensure base Maison Adrar perfume columns exist (older DBs may miss some)
    `ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS type TEXT;`,
    `ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS size TEXT;`,
    `ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS description TEXT;`,
    `ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS ingredients TEXT;`,
    `ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS discount NUMERIC(5,2);`,
    `ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS ean TEXT;`,
    `ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;`,
    `ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS sort_order INTEGER DEFAULT 0;`,

    // Extend Maison Adrar perfumes with richer schema
		`ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS gender_category TEXT;`,
		`ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS concentration TEXT;`,
		`ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS fragrance_family TEXT;`,
		`ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS top_notes TEXT;`,
		`ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS middle_notes TEXT;`,
		`ALTER TABLE maison_adrar_perfumes ADD COLUMN IF NOT EXISTS base_notes TEXT;`,

		// Extend perfume images
		`ALTER TABLE maison_adrar_perfume_images ADD COLUMN IF NOT EXISTS is_main BOOLEAN DEFAULT FALSE;`,

		// Extend perfume colors as variants
		`ALTER TABLE maison_adrar_perfume_colors ADD COLUMN IF NOT EXISTS price_override NUMERIC(12,2);`,
		`ALTER TABLE maison_adrar_perfume_colors ADD COLUMN IF NOT EXISTS volume_ml INTEGER;`,
		`ALTER TABLE maison_adrar_perfume_colors ADD COLUMN IF NOT EXISTS stock INTEGER;`,
		
		
		// Create address book tables
		`CREATE TABLE IF NOT EXISTS address_book (
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
		);`,
		
		`CREATE TABLE IF NOT EXISTS cities (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			name_ar TEXT,
			region TEXT NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);`,
		
		`CREATE TABLE IF NOT EXISTS quartiers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			city_id UUID NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			name_ar TEXT,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);`,
		
		`CREATE TABLE IF NOT EXISTS wishlist_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			product_id UUID NOT NULL REFERENCES product_models(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(user_id, product_id)
		);`,
		
		`CREATE INDEX IF NOT EXISTS idx_wishlist_items_user_id ON wishlist_items(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_wishlist_items_product_id ON wishlist_items(product_id);`,
		`CREATE INDEX IF NOT EXISTS idx_wishlist_items_created_at ON wishlist_items(created_at);`,
		
		// Promotional codes tables
		`CREATE TABLE IF NOT EXISTS promotional_codes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			code VARCHAR(50) UNIQUE NOT NULL,
			description TEXT,
			discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('percentage', 'fixed')),
			discount_value DECIMAL(10,2) NOT NULL,
			min_order_amount DECIMAL(10,2) DEFAULT 0,
			max_discount DECIMAL(10,2),
			usage_limit INTEGER DEFAULT -1,
			used_count INTEGER DEFAULT 0,
			is_active BOOLEAN DEFAULT true,
			start_date TIMESTAMP WITH TIME ZONE NOT NULL,
			expiry_date TIMESTAMP WITH TIME ZONE NOT NULL,
			created_by UUID REFERENCES users(id),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS promotional_code_usage (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			promotional_code_id UUID NOT NULL REFERENCES promotional_codes(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
			discount_amount DECIMAL(10,2) NOT NULL,
			used_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_promotional_codes_code ON promotional_codes(code);`,
		`CREATE INDEX IF NOT EXISTS idx_promotional_codes_active ON promotional_codes(is_active);`,
		`CREATE INDEX IF NOT EXISTS idx_promotional_codes_expiry ON promotional_codes(expiry_date);`,
		`CREATE INDEX IF NOT EXISTS idx_promotional_code_usage_code ON promotional_code_usage(promotional_code_id);`,
		`CREATE INDEX IF NOT EXISTS idx_promotional_code_usage_user ON promotional_code_usage(user_id);`,
		
		// Add external_code column to categories table if it doesn't exist
		`ALTER TABLE categories ADD COLUMN IF NOT EXISTS external_code TEXT;`,
		
		// Update existing users to have 'user' role if they don't have one
		`UPDATE users SET role = 'user' WHERE role IS NULL OR role = '';`,
		
		// Update existing users to have is_active = true if they don't have it set
		`UPDATE users SET is_active = TRUE WHERE is_active IS NULL;`,
		
		// Generate avatars for existing users who don't have one
		`UPDATE users SET avatar = 'https://api.dicebear.com/7.x/avataaars/svg?seed=' || id 
		 WHERE avatar IS NULL OR avatar = '';`,
		
		// Add missing columns to loyalty_accounts table if they don't exist
		`ALTER TABLE loyalty_accounts ADD COLUMN IF NOT EXISTS total_earned BIGINT DEFAULT 0;`,
		`ALTER TABLE loyalty_accounts ADD COLUMN IF NOT EXISTS total_redeemed BIGINT DEFAULT 0;`,
		
		// Add brand visual fields if they don't exist
		`ALTER TABLE brands ADD COLUMN IF NOT EXISTS logo TEXT;`,
		`ALTER TABLE brands ADD COLUMN IF NOT EXISTS banner TEXT;`,
		`ALTER TABLE brands ADD COLUMN IF NOT EXISTS color TEXT;`,
		
		// Add parent_category_id to brands table if it doesn't exist
		`ALTER TABLE brands ADD COLUMN IF NOT EXISTS parent_category_id UUID REFERENCES categories(id) ON DELETE SET NULL;`,
		
		// POS: augment orders table for POS source and payments
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS source TEXT DEFAULT 'web';`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_method_id UUID;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS tendered_amount NUMERIC(12,2) DEFAULT 0;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS change_due NUMERIC(12,2) DEFAULT 0;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS pos_agent_id UUID;`,
		
		// Add missing columns to orders table
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_option VARCHAR(20) DEFAULT 'delivery';`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_address JSONB;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_proof TEXT;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS promotional_code VARCHAR(50);`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(12,2) DEFAULT 0;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS currency VARCHAR(3) DEFAULT 'MRU';`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_status VARCHAR(20) DEFAULT 'pending';`,
		
		// Add delivery zone columns to orders table
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_zone_quartier_id VARCHAR(50);`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_zone_quartier_name VARCHAR(100);`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_zone_fee NUMERIC(12,2) DEFAULT 0;`,
		
		// Add missing columns to users table
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name VARCHAR(100);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name VARCHAR(100);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS push_token TEXT;`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT now();`,
		`ALTER TABLE quartiers ADD COLUMN IF NOT EXISTS delivery_fee DECIMAL(10,2) DEFAULT 0.00;`,
		
		// Add constraints for orders table
		`ALTER TABLE orders ADD CONSTRAINT IF NOT EXISTS chk_delivery_option CHECK (delivery_option IN ('pickup', 'delivery'));`,
		
		// Add missing columns to order_items table
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS product_id UUID REFERENCES product_models(id) ON DELETE CASCADE;`,
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS sku_id UUID REFERENCES skus(id) ON DELETE CASCADE;`,
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS quantity INTEGER NOT NULL DEFAULT 1;`,
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS unit_price NUMERIC(12,2) NOT NULL DEFAULT 0;`,
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS total_price NUMERIC(12,2) NOT NULL DEFAULT 0;`,
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS size VARCHAR(50);`,
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS color VARCHAR(50);`,
		`ALTER TABLE order_items ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();`,
		
		// Create payment_methods table if it doesn't exist
		`CREATE TABLE IF NOT EXISTS payment_methods (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			label TEXT NOT NULL,
			description TEXT,
			logo TEXT,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);`,
        // Inventory improvements
        `ALTER TABLE inventory ADD COLUMN IF NOT EXISTS reorder_point INT NOT NULL DEFAULT 0;`,
		
		// Create product views tracking table
		`CREATE TABLE IF NOT EXISTS product_views (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			product_id UUID NOT NULL REFERENCES product_models(id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id) ON DELETE SET NULL,
			anonymous_session_id VARCHAR(255),
			view_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			ip_address INET,
			user_agent TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		
		// Indexes for product views performance
		`CREATE INDEX IF NOT EXISTS idx_product_views_product_id ON product_views(product_id);`,
		`CREATE INDEX IF NOT EXISTS idx_product_views_user_id ON product_views(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_product_views_anonymous_session ON product_views(anonymous_session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_product_views_timestamp ON product_views(view_timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_product_views_composite ON product_views(product_id, view_timestamp);`,
		
		// Create an admin user if none exists
		`INSERT INTO users (id, phone, full_name, role, is_active, created_at, metadata) 
		 VALUES (gen_random_uuid(), '+22212345678', 'Admin User', 'admin', true, now(), '{}')
		 ON CONFLICT (phone) DO NOTHING;`,
	}

	for i, migration := range migrations {
		log.Printf("Running migration %d", i+1)
		if _, err := db.Exec(migration); err != nil {
			log.Printf("Warning: Migration %d failed: %v", i+1, err)
			// Continue with other migrations even if one fails
		}
	}

	log.Println("Migrations completed!")
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
