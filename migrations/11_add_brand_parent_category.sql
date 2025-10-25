-- Add parent_category_id column to brands table
ALTER TABLE brands ADD COLUMN IF NOT EXISTS parent_category_id UUID REFERENCES categories(id) ON DELETE SET NULL;
