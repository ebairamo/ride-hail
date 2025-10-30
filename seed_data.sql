-- Вставка пользователей
INSERT INTO users (id, email, role, status, password_hash) VALUES 
('11111111-1111-1111-1111-111111111111', 'admin@example.com', 'ADMIN', 'ACTIVE', 'password_hash_placeholder'),
('22222222-2222-2222-2222-222222222222', 'driver@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('33333333-3333-3333-3333-333333333333', 'passenger@example.com', 'PASSENGER', 'ACTIVE', 'password_hash_placeholder');

-- Вставка водителей
INSERT INTO drivers (id, license_number, vehicle_type, vehicle_attrs, rating, total_rides, total_earnings, status, is_verified) VALUES
('22222222-2222-2222-2222-222222222222', 'DL123456', 'ECONOMY', '{"vehicle_make": "Toyota", "vehicle_model": "Camry", "vehicle_color": "White", "vehicle_plate": "KZ 123 ABC"}', 4.8, 150, 250000.50, 'AVAILABLE', true);

-- Вставка координат
INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, fare_amount, distance_km, duration_minutes, is_current) VALUES
('22222222-2222-2222-2222-222222222222', 'driver', 'Almaty Central Park', 43.238949, 76.889709, NULL, NULL, NULL, true),
('33333333-3333-3333-3333-333333333333', 'passenger', 'Kok-Tobe Hill', 43.222015, 76.851511, NULL, NULL, NULL, true);

-- Вставка поездок
INSERT INTO rides (id, ride_number, passenger_id, driver_id, vehicle_type, status, estimated_fare, pickup_coordinate_id, destination_coordinate_id, requested_at) VALUES
('44444444-4444-4444-4444-444444444444', 'RIDE_20251016_001', '33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'ECONOMY', 'IN_PROGRESS', 1450.0, 
    (SELECT id FROM coordinates WHERE entity_id = '33333333-3333-3333-3333-333333333333' LIMIT 1),
    (SELECT id FROM coordinates WHERE entity_id = '22222222-2222-2222-2222-222222222222' LIMIT 1),
    NOW() - INTERVAL '30 minutes'
);

-- Добавим еще несколько поездок в разных статусах
INSERT INTO rides (id, ride_number, passenger_id, vehicle_type, status, estimated_fare, requested_at) VALUES
('55555555-5555-5555-5555-555555555555', 'RIDE_20251016_002', '33333333-3333-3333-3333-333333333333', 'PREMIUM', 'REQUESTED', 2100.0, NOW() - INTERVAL '10 minutes');

-- Завершенная поездка
INSERT INTO rides (id, ride_number, passenger_id, driver_id, vehicle_type, status, estimated_fare, final_fare, requested_at, completed_at) VALUES
('66666666-6666-6666-6666-666666666666', 'RIDE_20251016_003', '33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'ECONOMY', 'COMPLETED', 1350.0, 1400.0, 
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '1 hour 30 minutes'
);

-- История местоположений
INSERT INTO location_history (driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees, recorded_at, ride_id) VALUES
('22222222-2222-2222-2222-222222222222', 43.238949, 76.889709, 5.0, 45.0, 180.0, NOW() - INTERVAL '25 minutes', '44444444-4444-4444-4444-444444444444'),
('22222222-2222-2222-2222-222222222222', 43.235000, 76.880000, 4.0, 50.0, 175.0, NOW() - INTERVAL '20 minutes', '44444444-4444-4444-4444-444444444444'),
('22222222-2222-2222-2222-222222222222', 43.230000, 76.870000, 4.5, 40.0, 190.0, NOW() - INTERVAL '15 minutes', '44444444-4444-4444-4444-444444444444'),
('22222222-2222-2222-2222-222222222222', 43.225000, 76.860000, 3.0, 35.0, 185.0, NOW() - INTERVAL '10 minutes', '44444444-4444-4444-4444-444444444444'),
('22222222-2222-2222-2222-222222222222', 43.224000, 76.855000, 3.5, 20.0, 180.0, NOW() - INTERVAL '5 minutes', '44444444-4444-4444-4444-444444444444');