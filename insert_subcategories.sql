-- Insert real subcategories
INSERT INTO categories (id, name, slug, parent_id, level, is_active, created_at, updated_at) VALUES
('femme-robes', 'Robes', 'femme-robes', '98b91717-fb8d-4de5-a4bc-c97e2779f59c', 2, true, NOW(), NOW()),
('femme-tops', 'Tops & T-shirts', 'femme-tops', '98b91717-fb8d-4de5-a4bc-c97e2779f59c', 2, true, NOW(), NOW()),
('femme-pantalons', 'Pantalons', 'femme-pantalons', '98b91717-fb8d-4de5-a4bc-c97e2779f59c', 2, true, NOW(), NOW()),
('femme-chaussures', 'Chaussures', 'femme-chaussures', '98b91717-fb8d-4de5-a4bc-c97e2779f59c', 2, true, NOW(), NOW()),
('femme-accessoires', 'Accessoires', 'femme-accessoires', '98b91717-fb8d-4de5-a4bc-c97e2779f59c', 2, true, NOW(), NOW()),
('femme-sacs', 'Sacs', 'femme-sacs', '98b91717-fb8d-4de5-a4bc-c97e2779f59c', 2, true, NOW(), NOW()),
('homme-chemises', 'Chemises', 'homme-chemises', '0e75cf32-adc9-42c4-bc93-7503dffed911', 2, true, NOW(), NOW()),
('homme-t-shirts', 'T-shirts', 'homme-t-shirts', '0e75cf32-adc9-42c4-bc93-7503dffed911', 2, true, NOW(), NOW()),
('homme-pantalons', 'Pantalons', 'homme-pantalons', '0e75cf32-adc9-42c4-bc93-7503dffed911', 2, true, NOW(), NOW()),
('homme-chaussures', 'Chaussures', 'homme-chaussures', '0e75cf32-adc9-42c4-bc93-7503dffed911', 2, true, NOW(), NOW()),
('homme-accessoires', 'Accessoires', 'homme-accessoires', '0e75cf32-adc9-42c4-bc93-7503dffed911', 2, true, NOW(), NOW()),
('homme-montres', 'Montres', 'homme-montres', '0e75cf32-adc9-42c4-bc93-7503dffed911', 2, true, NOW(), NOW()),
('parfume-femme', 'Parfums Femme', 'parfume-femme', 'da8dcf81-3945-491b-bb29-4fcd37b65b0e', 2, true, NOW(), NOW()),
('parfume-homme', 'Parfums Homme', 'parfume-homme', 'da8dcf81-3945-491b-bb29-4fcd37b65b0e', 2, true, NOW(), NOW()),
('parfume-unisexe', 'Parfums Unisexe', 'parfume-unisexe', 'da8dcf81-3945-491b-bb29-4fcd37b65b0e', 2, true, NOW(), NOW()),
('parfume-maison', 'Parfums Maison', 'parfume-maison', 'da8dcf81-3945-491b-bb29-4fcd37b65b0e', 2, true, NOW(), NOW()),
('parfume-miniatures', 'Miniatures', 'parfume-miniatures', 'da8dcf81-3945-491b-bb29-4fcd37b65b0e', 2, true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
