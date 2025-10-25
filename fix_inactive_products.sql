-- Fix products that were accidentally set to inactive
-- This script reactivates all products that are currently inactive

UPDATE product_models 
SET is_active = true 
WHERE is_active = false;

-- Show the results
SELECT id, title, is_active, created_at 
FROM product_models 
ORDER BY created_at DESC;
