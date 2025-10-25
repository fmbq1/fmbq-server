-- Fix orders table schema to match the model
-- Add missing columns to orders table

-- Add delivery_option column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'orders' AND column_name = 'delivery_option') THEN
        ALTER TABLE orders ADD COLUMN delivery_option VARCHAR(20) NOT NULL DEFAULT 'delivery';
    END IF;
END $$;

-- Add delivery_address column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'orders' AND column_name = 'delivery_address') THEN
        ALTER TABLE orders ADD COLUMN delivery_address JSONB;
    END IF;
END $$;

-- Add payment_proof column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'orders' AND column_name = 'payment_proof') THEN
        ALTER TABLE orders ADD COLUMN payment_proof TEXT;
    END IF;
END $$;

-- Add promotional_code column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'orders' AND column_name = 'promotional_code') THEN
        ALTER TABLE orders ADD COLUMN promotional_code VARCHAR(50);
    END IF;
END $$;

-- Add discount_amount column if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'orders' AND column_name = 'discount_amount') THEN
        ALTER TABLE orders ADD COLUMN discount_amount NUMERIC(12,2) DEFAULT 0;
    END IF;
END $$;

-- Add constraints
ALTER TABLE orders ADD CONSTRAINT IF NOT EXISTS chk_delivery_option 
    CHECK (delivery_option IN ('pickup', 'delivery'));

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_orders_delivery_option ON orders(delivery_option);
CREATE INDEX IF NOT EXISTS idx_orders_promotional_code ON orders(promotional_code);

-- Add comments
COMMENT ON COLUMN orders.delivery_option IS 'Delivery method: pickup or delivery';
COMMENT ON COLUMN orders.delivery_address IS 'JSON object containing delivery address details';
COMMENT ON COLUMN orders.payment_proof IS 'URL or path to payment proof image';
COMMENT ON COLUMN orders.promotional_code IS 'Promotional code used for this order';
COMMENT ON COLUMN orders.discount_amount IS 'Total discount amount applied from promotional code';
