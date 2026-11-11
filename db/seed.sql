-- =============================================================
-- Тестовые данные для проекта «Каршеринг»
-- Запуск из корня проекта (после db/schema.sql):
--     psql -U carsharing -d carsharing -f db/seed.sql
-- Всё выполняется одной транзакцией: при ошибке данные не будут загружены.
--
-- Тестовые учётные записи (пароли только для разработки):
--   admin@carsharing.local     / Admin123!    — администратор
--   manager1@carsharing.local  / Manager123!  — менеджер автопарка
--   manager2@carsharing.local  / Manager123!  — менеджер автопарка
--   клиенты (все с паролем Client123!):
--     ivan.petrov@example.com, anna.sidorova@example.com,
--     dmitry.ivanov@example.com, maria.volkova@example.com,
--     oleg.novikov@example.com
--     new.client@example.com — без водительского удостоверения
--
-- Пароли хранятся как bcrypt-хеши (cost 10), формат совместим с Go (golang.org/x/crypto/bcrypt).
-- =============================================================

\set ON_ERROR_STOP on
SET client_encoding = 'UTF8';

BEGIN;

TRUNCATE maintenance_logs, rentals, cars, users, tariffs RESTART IDENTITY CASCADE;

-- ---------- tariffs ----------
INSERT INTO tariffs (name, price_per_min, waiting_per_min) VALUES
    ('Эконом', 8.90, 2.00),
    ('Комфорт', 12.90, 3.00),
    ('Бизнес', 19.90, 5.00);

-- ---------- users ----------
INSERT INTO users (email, password_hash, full_name, driver_license, role) VALUES
    ('admin@carsharing.local', '$2b$10$OV6aj6V8wdaAJFiEOWr7RevIQLazCggxVQvOydOtcdkyq97QRk09u', 'Администратор Системы', NULL, 'admin'),
    ('manager1@carsharing.local', '$2b$10$6phnxN11PAtqxch8TPMS7uMVOCmNKzlW6KzQMzy4qAHkTLFshjMaC', 'Смирнов Алексей Петрович', NULL, 'manager'),
    ('manager2@carsharing.local', '$2b$10$h1C77jzup4.yBovrUzlOG.jG4BLHTqZrSQbhSMeAsU8qmgs1ycoIW', 'Кузнецова Елена Андреевна', NULL, 'manager'),
    ('ivan.petrov@example.com', '$2b$10$phmzkJ45HB9CrehS61MakeYJjHbBapcE.5wT8rBpyaxKDFDjVqO0S', 'Петров Иван Сергеевич', '7712 345678', 'client'),
    ('anna.sidorova@example.com', '$2b$10$wZ653QgOZ4Tos8gNpBTgeO/g0kYUWfrcZItGimDEm14tqLgWxMtaa', 'Сидорова Анна Викторовна', '7708 112233', 'client'),
    ('dmitry.ivanov@example.com', '$2b$10$QnzJUDBwIYkAEEXGzOeryeaOTRu2fCATID5glV4lvG1OKNSTpO4Ym', 'Иванов Дмитрий Олегович', '7715 998877', 'client'),
    ('maria.volkova@example.com', '$2b$10$1MsNBynYN7uw/tDISewphuunqcpUSjFn/6txhhH9SbAEdXvEImZt.', 'Волкова Мария Игоревна', '7704 556677', 'client'),
    ('oleg.novikov@example.com', '$2b$10$FCCu.oDzxw4qRSPe6804JO421qdYZScmj5H1BjnWYdmXa34mNWjQa', 'Новиков Олег Павлович', '7719 223344', 'client'),
    ('new.client@example.com', '$2b$10$PJixzUPUFwIpjJ8SrZwMyO89LEiQkRpMOgdWzAA1zYpAznIqspzIq', 'Соколов Артём Николаевич', NULL, 'client');

-- ---------- cars ----------
-- Тариф подбирается по названию; координаты — центр Москвы (для карты Leaflet)
INSERT INTO cars (tariff_id, brand_model, plate_number, fuel_level, latitude, longitude, status)
SELECT t.id, v.brand_model, v.plate_number, v.fuel_level, v.latitude, v.longitude, v.status
FROM (VALUES
    ('Эконом', 'Kia Rio', 'А101АА777', 78, 55.7558, 37.6173, 'available'),
    ('Эконом', 'Hyundai Solaris', 'В202ВВ777', 54, 55.7612, 37.5891, 'available'),
    ('Эконом', 'Volkswagen Polo', 'Е303ЕЕ777', 91, 55.7405, 37.642, 'in_use'),
    ('Эконом', 'Skoda Rapid', 'К404КК777', 33, 55.7701, 37.6553, 'booked'),
    ('Эконом', 'Renault Logan', 'М505ММ777', 66, 55.7289, 37.5978, 'available'),
    ('Эконом', 'Lada Vesta', 'Н606НН777', 12, 55.7822, 37.6012, 'available'),
    ('Эконом', 'Kia Rio', 'О707ОО777', 87, 55.7143, 37.6205, 'maintenance'),
    ('Эконом', 'Hyundai Solaris', 'Р808РР777', 45, 55.7945, 37.5762, 'decommissioned'),
    ('Комфорт', 'Toyota Camry', 'С909СС777', 82, 55.7497, 37.5395, 'available'),
    ('Комфорт', 'Kia K5', 'Т111ТТ777', 59, 55.7667, 37.6198, 'in_use'),
    ('Комфорт', 'Hyundai Sonata', 'У222УУ777', 71, 55.7352, 37.6789, 'available'),
    ('Комфорт', 'Skoda Octavia', 'Х333ХХ777', 28, 55.7589, 37.6641, 'booked'),
    ('Комфорт', 'Volkswagen Tiguan', 'А444АА777', 95, 55.781, 37.633, 'available'),
    ('Комфорт', 'Toyota RAV4', 'В555ВВ777', 63, 55.7036, 37.5832, 'available'),
    ('Бизнес', 'Mercedes-Benz E-Class', 'Е666ЕЕ777', 88, 55.7516, 37.5993, 'available'),
    ('Бизнес', 'BMW 5 Series', 'К777КК777', 74, 55.7634, 37.6405, 'available'),
    ('Бизнес', 'Audi A6', 'М888ММ777', 39, 55.7301, 37.6104, 'available'),
    ('Бизнес', 'Genesis G80', 'Н999НН777', 100, 55.7728, 37.5871, 'available')
) AS v(tariff_name, brand_model, plate_number, fuel_level, latitude, longitude, status)
JOIN tariffs t ON t.name = v.tariff_name;

-- ---------- rentals ----------
-- Тариф и цены копируются из автомобиля (снимок на момент создания аренды).
-- total_cost для завершённых аренд: цена_минуты * минуты_поездки + цена_ожидания * минуты_ожидания
-- (минуты ожидания = от бронирования до старта). Это упрощённая формула для тестовых данных;
-- итоговая логика расчёта будет в слое services на Go.
INSERT INTO rentals (user_id, tariff_id, car_id, booked_at, started_at, ended_at,
                     price_per_min, waiting_per_min, total_cost, status)
SELECT u.id, t.id, c.id, v.booked_at, v.started_at, v.ended_at,
       t.price_per_min, t.waiting_per_min,
       CASE WHEN v.status = 'completed' THEN
            round((t.price_per_min   * extract(epoch FROM v.ended_at   - v.started_at) / 60
                 + t.waiting_per_min * extract(epoch FROM v.started_at - v.booked_at)  / 60)::numeric, 2)
       END,
       v.status
FROM (VALUES
    ('ivan.petrov@example.com', 'А101АА777', timestamptz '2026-10-20 09:50+03', timestamptz '2026-10-20 10:00+03', timestamptz '2026-10-20 10:35+03', 'completed'),
    ('anna.sidorova@example.com', 'С909СС777', timestamptz '2026-10-22 18:10+03', timestamptz '2026-10-22 18:17+03', timestamptz '2026-10-22 19:05+03', 'completed'),
    ('dmitry.ivanov@example.com', 'В202ВВ777', timestamptz '2026-10-25 12:00+03', timestamptz '2026-10-25 12:05+03', timestamptz '2026-10-25 12:50+03', 'completed'),
    ('maria.volkova@example.com', 'Е666ЕЕ777', timestamptz '2026-10-28 20:30+03', timestamptz '2026-10-28 20:48+03', timestamptz '2026-10-28 22:10+03', 'completed'),
    ('ivan.petrov@example.com', 'У222УУ777', timestamptz '2026-11-01 08:40+03', timestamptz '2026-11-01 08:45+03', timestamptz '2026-11-01 09:20+03', 'completed'),
    ('oleg.novikov@example.com', 'М505ММ777', timestamptz '2026-11-03 14:00+03', timestamptz '2026-11-03 14:12+03', timestamptz '2026-11-03 15:30+03', 'completed'),
    ('anna.sidorova@example.com', 'К777КК777', timestamptz '2026-11-05 19:00+03', timestamptz '2026-11-05 19:06+03', timestamptz '2026-11-05 20:15+03', 'completed'),
    ('maria.volkova@example.com', 'А101АА777', timestamptz '2026-11-08 11:20+03', timestamptz '2026-11-08 11:25+03', timestamptz '2026-11-08 11:58+03', 'completed'),
    ('dmitry.ivanov@example.com', 'А444АА777', timestamptz '2026-11-06 10:00+03', NULL::timestamptz, NULL::timestamptz, 'cancelled'),
    ('oleg.novikov@example.com', 'М888ММ777', timestamptz '2026-11-09 16:30+03', NULL::timestamptz, NULL::timestamptz, 'cancelled'),
    ('ivan.petrov@example.com', 'Е303ЕЕ777', now() - interval '35 minutes', now() - interval '25 minutes', NULL::timestamptz, 'active'),
    ('anna.sidorova@example.com', 'Т111ТТ777', now() - interval '50 minutes', now() - interval '40 minutes', NULL::timestamptz, 'active'),
    ('maria.volkova@example.com', 'К404КК777', now() - interval '5 minutes', NULL::timestamptz, NULL::timestamptz, 'booked'),
    ('oleg.novikov@example.com', 'Х333ХХ777', now() - interval '2 minutes', NULL::timestamptz, NULL::timestamptz, 'booked')
) AS v(email, plate_number, booked_at, started_at, ended_at, status)
JOIN users   u ON u.email        = v.email
JOIN cars    c ON c.plate_number = v.plate_number
JOIN tariffs t ON t.id           = c.tariff_id;

-- ---------- maintenance_logs ----------
INSERT INTO maintenance_logs (car_id, manager_id, description, cost, logged_at)
SELECT c.id, u.id, v.description, v.cost, v.logged_at
FROM (VALUES
    ('А101АА777', 'manager1@carsharing.local', 'Плановое ТО: замена масла и фильтров', 4500.00::numeric, timestamptz '2026-10-05 11:00+03'),
    ('Е303ЕЕ777', 'manager1@carsharing.local', 'Диагностика подвески', 2500.00::numeric, timestamptz '2026-10-12 15:30+03'),
    ('К777КК777', 'manager2@carsharing.local', 'Плановое ТО: замена масла и фильтров', 8900.00::numeric, timestamptz '2026-10-15 10:15+03'),
    ('Е666ЕЕ777', 'manager2@carsharing.local', 'Полировка кузова и химчистка салона', 7800.00::numeric, timestamptz '2026-10-18 13:00+03'),
    ('С909СС777', 'manager2@carsharing.local', 'Замена шин на зимние', 12000.00::numeric, timestamptz '2026-10-28 09:45+03'),
    ('Р808РР777', 'manager2@carsharing.local', 'Списание: ремонт двигателя экономически нецелесообразен', 0.00::numeric, timestamptz '2026-10-30 17:00+03'),
    ('Н606НН777', 'manager1@carsharing.local', 'Замена аккумулятора', 9500.00::numeric, timestamptz '2026-11-02 12:20+03'),
    ('О707ОО777', 'manager1@carsharing.local', 'Замена передних тормозных колодок', 6200.00::numeric, timestamptz '2026-11-10 10:00+03')
) AS v(plate_number, manager_email, description, cost, logged_at)
JOIN cars  c ON c.plate_number = v.plate_number
JOIN users u ON u.email        = v.manager_email;

COMMIT;

-- Краткая сводка после загрузки
SELECT 'tariffs' AS table_name, count(*) FROM tariffs
UNION ALL SELECT 'users',            count(*) FROM users
UNION ALL SELECT 'cars',             count(*) FROM cars
UNION ALL SELECT 'rentals',          count(*) FROM rentals
UNION ALL SELECT 'maintenance_logs', count(*) FROM maintenance_logs;
