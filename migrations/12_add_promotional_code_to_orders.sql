-- Add promotional code support to orders table
ALTER TABLE orders 
ADD COLUMN IF NOT EXISTS promotional_code VARCHAR(50),
ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(12,2) DEFAULT 0;

-- Add index for promotional code lookups
CREATE INDEX IF NOT EXISTS idx_orders_promotional_code ON orders(promotional_code);

-- Add comment for documentation
COMMENT ON COLUMN orders.promotional_code IS 'Promotional code used for this order';
COMMENT ON COLUMN orders.discount_amount IS 'Total discount amount applied from promotional code';
