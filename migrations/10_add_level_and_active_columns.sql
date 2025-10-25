-- Add missing columns to categories table
ALTER TABLE categories ADD COLUMN IF NOT EXISTS level INTEGER DEFAULT 1;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Update existing categories to have level 1
UPDATE categories SET level = 1 WHERE parent_id IS NULL;
UPDATE categories SET level = 2 WHERE parent_id IS NOT NULL;

-- Create index on level column
CREATE INDEX IF NOT EXISTS idx_categories_level ON categories(level);
