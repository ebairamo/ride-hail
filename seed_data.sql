begin;

-- Вставка пользователей (соответствует таблице users)
INSERT INTO users (id, email, role, status, password_hash) VALUES 
('11111111-1111-1111-1111-111111111111', 'admin@example.com', 'ADMIN', 'ACTIVE', 'password_hash_placeholder'),
('22222222-2222-2222-2222-222222222222', 'driver1@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('33333333-3333-3333-3333-333333333333', 'passenger1@example.com', 'PASSENGER', 'ACTIVE', 'password_hash_placeholder'),
('44444444-4444-4444-4444-444444444444', 'driver2@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('55555555-5555-5555-5555-555555555555', 'driver3@example.com', 'DRIVER', 'ACTIVE', 'password_hash_placeholder'),
('66666666-6666-6666-6666-666666666666', 'passenger2@example.com', 'PASSENGER', 'ACTIVE', 'password_hash_placeholder');

-- Вставка координат (соответствует таблице coordinates)
INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current) VALUES
-- Координаты водителей
('22222222-2222-2222-2222-222222222222', 'driver', 'Almaty Central Park', 43.238949, 76.889709, true),
('44444444-4444-4444-4444-444444444444', 'driver', 'Abay Ave, near Kazakhstan Hotel', 43.250000, 76.950000, true),
('55555555-5555-5555-5555-555555555555', 'driver', 'Al-Farabi District', 43.230000, 76.920000, true),
-- Координаты пассажиров
('33333333-3333-3333-3333-333333333333', 'passenger', 'Kok-Tobe Hill', 43.222015, 76.851511, true),
('66666666-6666-6666-6666-666666666666', 'passenger', 'Al-Farabi Ave, Mega Center', 43.220000, 76.910000, true);

-- Инициализация счетчика поездок (соответствует таблице ride_counters)
INSERT INTO ride_counters (ride_date, counter) VALUES 
(CURRENT_DATE, 100);

-- Вставка поездок (соответствует таблице rides)
INSERT INTO rides (id, ride_number, passenger_id, driver_id, vehicle_type, status, estimated_fare, final_fare, requested_at, matched_at, started_at, completed_at, pickup_coordinate_id, destination_coordinate_id) VALUES
-- Активная поездка
('77777777-7777-7777-7777-777777777777', 'RIDE_20251016_101', '33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'ECONOMY', 'IN_PROGRESS', 1450.0, NULL, 
    NOW() - INTERVAL '30 minutes',
    NOW() - INTERVAL '25 minutes',
    NOW() - INTERVAL '20 minutes',
    NULL,
    (SELECT id FROM coordinates WHERE entity_id = '33333333-3333-3333-3333-333333333333' AND entity_type = 'passenger' LIMIT 1),
    (SELECT id FROM coordinates WHERE entity_id = '22222222-2222-2222-2222-222222222222' AND entity_type = 'driver' LIMIT 1)
),

-- Запрошенная поездка (ищет водителя)
('88888888-8888-8888-8888-888888888888', 'RIDE_20251016_102', '66666666-6666-6666-6666-666666666666', NULL, 'PREMIUM', 'REQUESTED', 2100.0, NULL,
    NOW() - INTERVAL '10 minutes',
    NULL, NULL, NULL,
    (SELECT id FROM coordinates WHERE entity_id = '66666666-6666-6666-6666-666666666666' AND entity_type = 'passenger' LIMIT 1),
    (SELECT id FROM coordinates WHERE entity_id = '44444444-4444-4444-4444-444444444444' AND entity_type = 'driver' LIMIT 1)
),

-- Завершенная поездка
('99999999-9999-9999-9999-999999999999', 'RIDE_20251016_103', '33333333-3333-3333-3333-333333333333', '55555555-5555-5555-5555-555555555555', 'XL', 'COMPLETED', 1800.0, 1750.0,
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '1 hour 55 minutes',
    NOW() - INTERVAL '1 hour 50 minutes',
    NOW() - INTERVAL '1 hour 30 minutes',
    (SELECT id FROM coordinates WHERE entity_id = '33333333-3333-3333-3333-333333333333' AND entity_type = 'passenger' LIMIT 1),
    (SELECT id FROM coordinates WHERE entity_id = '55555555-5555-5555-5555-555555555555' AND entity_type = 'driver' LIMIT 1)
);

-- Вставка событий поездок (соответствует таблице ride_events)
INSERT INTO ride_events (ride_id, event_type, event_data) VALUES
-- События для активной поездки
('77777777-7777-7777-7777-777777777777', 'RIDE_REQUESTED', '{"passenger_id": "33333333-3333-3333-3333-333333333333", "vehicle_type": "ECONOMY", "estimated_fare": 1450.0}'),
('77777777-7777-7777-7777-777777777777', 'DRIVER_MATCHED', '{"driver_id": "22222222-2222-2222-2222-222222222222", "old_status": "REQUESTED", "new_status": "MATCHED"}'),
('77777777-7777-7777-7777-777777777777', 'DRIVER_ARRIVED', '{"location": {"lat": 43.222015, "lng": 76.851511}}'),
('77777777-7777-7777-7777-777777777777', 'RIDE_STARTED', '{"old_status": "ARRIVED", "new_status": "IN_PROGRESS"}'),

-- События для запрошенной поездки
('88888888-8888-8888-8888-888888888888', 'RIDE_REQUESTED', '{"passenger_id": "66666666-6666-6666-6666-666666666666", "vehicle_type": "PREMIUM", "estimated_fare": 2100.0}'),

-- События для завершенной поездки
('99999999-9999-9999-9999-999999999999', 'RIDE_REQUESTED', '{"passenger_id": "33333333-3333-3333-3333-333333333333", "vehicle_type": "XL", "estimated_fare": 1800.0}'),
('99999999-9999-9999-9999-999999999999', 'DRIVER_MATCHED', '{"driver_id": "55555555-5555-5555-5555-555555555555", "old_status": "REQUESTED", "new_status": "MATCHED"}'),
('99999999-9999-9999-9999-999999999999', 'RIDE_STARTED', '{"old_status": "ARRIVED", "new_status": "IN_PROGRESS"}'),
('99999999-9999-9999-9999-999999999999', 'RIDE_COMPLETED', '{"final_fare": 1750.0, "old_status": "IN_PROGRESS", "new_status": "COMPLETED"}');

commit;