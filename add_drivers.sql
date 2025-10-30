BEGIN;

-- Создаем 10 новых пользователей (водителей)
INSERT INTO users (id, email, role, status, password_hash) VALUES 
('a1111111-1111-1111-1111-111111111111', 'driver_eco1@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a2222222-2222-2222-2222-222222222222', 'driver_eco2@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a3333333-3333-3333-3333-333333333333', 'driver_eco3@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a4444444-4444-4444-4444-444444444444', 'driver_eco4@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a5555555-5555-5555-5555-555555555555', 'driver_prem1@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a6666666-6666-6666-6666-666666666666', 'driver_prem2@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a7777777-7777-7777-7777-777777777777', 'driver_prem3@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a8888888-8888-8888-8888-888888888888', 'driver_xl1@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a9999999-9999-9999-9999-999999999999', 'driver_xl2@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('a0000000-0000-0000-0000-000000000000', 'driver_xl3@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder');

-- Создаем водителей с разными типами автомобилей и статусами
INSERT INTO drivers (id, license_number, vehicle_type, vehicle_attrs, status, rating, total_rides, total_earnings, is_verified) VALUES 
-- ECONOMY водители (4 штуки: 2 доступных, 1 занятый, 1 оффлайн)
('a1111111-1111-1111-1111-111111111111', 'ECO-001', 'ECONOMY', 
  '{"vehicle_make": "Toyota", "vehicle_model": "Corolla", "vehicle_color": "Silver", "vehicle_plate": "A123ABC", "vehicle_year": 2020}'::jsonb, 
  'AVAILABLE', 4.8, 120, 150000.00, true),

('a2222222-2222-2222-2222-222222222222', 'ECO-002', 'ECONOMY', 
  '{"vehicle_make": "Hyundai", "vehicle_model": "Elantra", "vehicle_color": "Blue", "vehicle_plate": "B234BCD", "vehicle_year": 2019}'::jsonb, 
  'AVAILABLE', 4.5, 80, 100000.00, true),

('a3333333-3333-3333-3333-333333333333', 'ECO-003', 'ECONOMY', 
  '{"vehicle_make": "Nissan", "vehicle_model": "Sentra", "vehicle_color": "White", "vehicle_plate": "C345CDE", "vehicle_year": 2021}'::jsonb, 
  'BUSY', 4.7, 95, 120000.00, true),

('a4444444-4444-4444-4444-444444444444', 'ECO-004', 'ECONOMY', 
  '{"vehicle_make": "Honda", "vehicle_model": "Civic", "vehicle_color": "Black", "vehicle_plate": "D456DEF", "vehicle_year": 2018}'::jsonb, 
  'OFFLINE', 4.6, 110, 140000.00, true),

-- PREMIUM водители (3 штуки: 2 доступных, 1 занятый)
('a5555555-5555-5555-5555-555555555555', 'PREM-001', 'PREMIUM', 
  '{"vehicle_make": "Mercedes-Benz", "vehicle_model": "E-Class", "vehicle_color": "Black", "vehicle_plate": "E567EFG", "vehicle_year": 2022}'::jsonb, 
  'AVAILABLE', 4.9, 150, 300000.00, true),

('a6666666-6666-6666-6666-666666666666', 'PREM-002', 'PREMIUM', 
  '{"vehicle_make": "BMW", "vehicle_model": "5 Series", "vehicle_color": "Silver", "vehicle_plate": "F678FGH", "vehicle_year": 2021}'::jsonb, 
  'AVAILABLE', 4.8, 140, 280000.00, true),

('a7777777-7777-7777-7777-777777777777', 'PREM-003', 'PREMIUM', 
  '{"vehicle_make": "Lexus", "vehicle_model": "ES", "vehicle_color": "White", "vehicle_plate": "G789GHI", "vehicle_year": 2022}'::jsonb, 
  'BUSY', 4.9, 160, 320000.00, true),

-- XL водители (3 штуки: 1 доступный, 1 занятый, 1 в пути)
('a8888888-8888-8888-8888-888888888888', 'XL-001', 'XL', 
  '{"vehicle_make": "Toyota", "vehicle_model": "Highlander", "vehicle_color": "Black", "vehicle_plate": "H890HIJ", "vehicle_year": 2021}'::jsonb, 
  'AVAILABLE', 4.7, 90, 200000.00, true),

('a9999999-9999-9999-9999-999999999999', 'XL-002', 'XL', 
  '{"vehicle_make": "Honda", "vehicle_model": "Pilot", "vehicle_color": "Blue", "vehicle_plate": "I901IJK", "vehicle_year": 2020}'::jsonb, 
  'BUSY', 4.8, 85, 190000.00, true),

('a0000000-0000-0000-0000-000000000000', 'XL-003', 'XL', 
  '{"vehicle_make": "Ford", "vehicle_model": "Explorer", "vehicle_color": "Red", "vehicle_plate": "J012JKL", "vehicle_year": 2022}'::jsonb, 
  'EN_ROUTE', 4.6, 70, 160000.00, true);

-- Добавляем текущую локацию для водителей
INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current) VALUES
('a1111111-1111-1111-1111-111111111111', 'driver', 'Almaty, Abay Avenue', 43.238000, 76.889000, true),
('a2222222-2222-2222-2222-222222222222', 'driver', 'Almaty, Dostyk Avenue', 43.240000, 76.891000, true),
('a3333333-3333-3333-3333-333333333333', 'driver', 'Almaty, Al-Farabi Avenue', 43.230000, 76.880000, true),
('a5555555-5555-5555-5555-555555555555', 'driver', 'Almaty, Tole Bi Street', 43.250000, 76.895000, true),
('a6666666-6666-6666-6666-666666666666', 'driver', 'Almaty, Seifullin Avenue', 43.245000, 76.893000, true),
('a7777777-7777-7777-7777-777777777777', 'driver', 'Almaty, Furmanov Street', 43.242000, 76.888000, true),
('a8888888-8888-8888-8888-888888888888', 'driver', 'Almaty, Timiryazev Street', 43.235000, 76.882000, true),
('a9999999-9999-9999-9999-999999999999', 'driver', 'Almaty, Zhibek Zholy', 43.255000, 76.900000, true),
('a0000000-0000-0000-0000-000000000000', 'driver', 'Almaty, Gogol Street', 43.248000, 76.897000, true);

COMMIT;