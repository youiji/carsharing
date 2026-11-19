-- =============================================================
-- Проверка целостности базы данных «Каршеринг»
-- Запуск из корня проекта (после db/schema.sql и db/seed.sql):
--     psql -U carsharing -d carsharing -f db/checks.sql
--
-- Каждый тест выполняется в своей транзакции и заканчивается ROLLBACK,
-- поэтому данные в базе не меняются.
-- В тестах 1–27 база ДОЛЖНА отклонить запрос: в выводе будет строка ERROR
-- с именем нарушенного ограничения (имена — из db/schema.sql).
-- В тестах 28–30 запрос должен ПРОЙТИ (INSERT 0 1 / UPDATE 1 / DELETE 1).
-- Сообщения об ошибках выводит сам PostgreSQL, на английском языке.
-- =============================================================

\set ON_ERROR_STOP off
SET client_encoding = 'UTF8';

\echo
\echo '=============== A. Нарушения ограничений аренд (rentals) ==============='

\echo
\echo 'Тест 1. Двойное бронирование: машина Е303ЕЕ777 уже в активной аренде'
\echo 'Ожидается: ERROR ... uq_rentals_one_active_per_car'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, t.price_per_min, t.waiting_per_min, 'booked'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'dmitry.ivanov@example.com' AND c.plate_number = 'Е303ЕЕ777';
ROLLBACK;

\echo
\echo 'Тест 2. Вторая аренда у клиента, у которого уже есть бронирование (oleg.novikov)'
\echo 'Ожидается: ERROR ... uq_rentals_one_active_per_user'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, t.price_per_min, t.waiting_per_min, 'booked'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'oleg.novikov@example.com' AND c.plate_number = 'А444АА777';
ROLLBACK;

\echo
\echo 'Тест 3. Статус completed без ended_at'
\echo 'Ожидается: ERROR ... chk_rentals_status_dates'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, booked_at, started_at, ended_at,
                     price_per_min, waiting_per_min, total_cost, status)
SELECT u.id, c.tariff_id, c.id, now() - interval '10 minutes', now(), NULL,
       t.price_per_min, t.waiting_per_min, 100, 'completed'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'oleg.novikov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo 'Тест 4. Статус active без started_at'
\echo 'Ожидается: ERROR ... chk_rentals_status_dates'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, t.price_per_min, t.waiting_per_min, 'active'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'oleg.novikov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo 'Тест 5. Статус completed без total_cost'
\echo 'Ожидается: ERROR ... chk_rentals_status_dates'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, booked_at, started_at, ended_at,
                     price_per_min, waiting_per_min, total_cost, status)
SELECT u.id, c.tariff_id, c.id, now() - interval '3 hours', now() - interval '2 hours',
       now() - interval '1 hour', t.price_per_min, t.waiting_per_min, NULL, 'completed'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'oleg.novikov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo 'Тест 6. Старт поездки раньше бронирования (started_at < booked_at)'
\echo 'Ожидается: ERROR ... chk_rentals_started_after_booked'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, booked_at, started_at,
                     price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, now(), now() - interval '1 hour',
       t.price_per_min, t.waiting_per_min, 'active'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'oleg.novikov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo 'Тест 7. Окончание поездки раньше старта (ended_at < started_at)'
\echo 'Ожидается: ERROR ... chk_rentals_ended_after_started'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, booked_at, started_at, ended_at,
                     price_per_min, waiting_per_min, total_cost, status)
SELECT u.id, c.tariff_id, c.id, now() - interval '3 hours', now() - interval '2 hours',
       now() - interval '150 minutes', t.price_per_min, t.waiting_per_min, 100, 'completed'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'oleg.novikov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo 'Тест 8. Неизвестный статус аренды'
\echo 'Ожидается: ERROR ... chk_rentals_status'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, t.price_per_min, t.waiting_per_min, 'paused'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'oleg.novikov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo 'Тест 9. Отрицательная цена в аренде'
\echo 'Ожидается: ERROR ... chk_rentals_price'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, -5, t.waiting_per_min, 'booked'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'oleg.novikov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo 'Тест 10. Бронирование с заполненной стоимостью'
\echo 'Ожидается: ERROR ... chk_rentals_status_dates'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, total_cost, status)
SELECT u.id, c.tariff_id, c.id, t.price_per_min, t.waiting_per_min, 100, 'booked'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'dmitry.ivanov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo 'Тест 11. Отмена бронирования без момента отмены и списанной суммы'
\echo 'Ожидается: ERROR ... chk_rentals_status_dates'
BEGIN;
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, t.price_per_min, t.waiting_per_min, 'cancelled'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'dmitry.ivanov@example.com' AND c.plate_number = 'В202ВВ777';
ROLLBACK;

\echo
\echo '=============== B. Нарушения ограничений автомобилей, пользователей, тарифов ==============='

\echo
\echo 'Тест 12. Уровень топлива 150 %'
\echo 'Ожидается: ERROR ... chk_cars_fuel_level'
BEGIN;
INSERT INTO cars (tariff_id, brand_model, plate_number, fuel_level, latitude, longitude)
SELECT id, 'Test Car', 'Т001ТТ799', 150, 55.75, 37.61 FROM tariffs WHERE name = 'Эконом';
ROLLBACK;

\echo
\echo 'Тест 13. Широта 95 (допустимо от -90 до 90)'
\echo 'Ожидается: ERROR ... chk_cars_latitude'
BEGIN;
INSERT INTO cars (tariff_id, brand_model, plate_number, fuel_level, latitude, longitude)
SELECT id, 'Test Car', 'Т002ТТ799', 50, 95, 37.61 FROM tariffs WHERE name = 'Эконом';
ROLLBACK;

\echo
\echo 'Тест 14. Долгота 200 (допустимо от -180 до 180)'
\echo 'Ожидается: ERROR ... chk_cars_longitude'
BEGIN;
INSERT INTO cars (tariff_id, brand_model, plate_number, fuel_level, latitude, longitude)
SELECT id, 'Test Car', 'Т003ТТ799', 50, 55.75, 200 FROM tariffs WHERE name = 'Эконом';
ROLLBACK;

\echo
\echo 'Тест 15. Неизвестный статус автомобиля'
\echo 'Ожидается: ERROR ... chk_cars_status'
BEGIN;
INSERT INTO cars (tariff_id, brand_model, plate_number, fuel_level, latitude, longitude, status)
SELECT id, 'Test Car', 'Т004ТТ799', 50, 55.75, 37.61, 'flying' FROM tariffs WHERE name = 'Эконом';
ROLLBACK;

\echo
\echo 'Тест 16. Дубликат гос. номера'
\echo 'Ожидается: ERROR ... uq_cars_plate_number'
BEGIN;
INSERT INTO cars (tariff_id, brand_model, plate_number, fuel_level, latitude, longitude)
SELECT id, 'Test Car', 'А101АА777', 50, 55.75, 37.61 FROM tariffs WHERE name = 'Эконом';
ROLLBACK;

\echo
\echo 'Тест 17. Неизвестная роль пользователя'
\echo 'Ожидается: ERROR ... chk_users_role'
BEGIN;
INSERT INTO users (email, password_hash, full_name, role)
VALUES ('hacker@example.com', 'x', 'Test User', 'superadmin');
ROLLBACK;

\echo
\echo 'Тест 18. Дубликат email в другом регистре (CITEXT)'
\echo 'Ожидается: ERROR ... uq_users_email'
BEGIN;
INSERT INTO users (email, password_hash, full_name)
VALUES ('IVAN.PETROV@EXAMPLE.COM', 'x', 'Test User');
ROLLBACK;

\echo
\echo 'Тест 19. Отрицательная цена тарифа'
\echo 'Ожидается: ERROR ... chk_tariffs_price'
BEGIN;
INSERT INTO tariffs (name, price_per_min, waiting_per_min) VALUES ('Тест', -1, 1);
ROLLBACK;

\echo
\echo 'Тест 20. Водительское удостоверение не в формате NNNN NNNNNN'
\echo 'Ожидается: ERROR ... chk_users_driver_license'
BEGIN;
INSERT INTO users (email, password_hash, full_name, driver_license)
VALUES ('license.test@example.com', 'x', 'Test User', '77AB-12');
ROLLBACK;

\echo
\echo 'Тест 21. Название тарифа, отличающееся от существующего только регистром'
\echo 'Ожидается: ERROR ... uq_tariffs_name'
BEGIN;
INSERT INTO tariffs (name, price_per_min, waiting_per_min) VALUES ('ЭКОНОМ', 1, 1);
ROLLBACK;

\echo
\echo 'Тест 22. Пустое ФИО пользователя'
\echo 'Ожидается: ERROR ... chk_users_full_name'
BEGIN;
INSERT INTO users (email, password_hash, full_name) VALUES ('empty.name@example.com', 'x', '   ');
ROLLBACK;

\echo
\echo 'Тест 23. Email длиннее 100 символов'
\echo 'Ожидается: ERROR ... chk_users_email_length'
BEGIN;
INSERT INTO users (email, password_hash, full_name) VALUES (repeat('a', 95) || '@example.com', 'x', 'Test User');
ROLLBACK;

\echo
\echo 'Тест 24. Запись обслуживания с пустым описанием'
\echo 'Ожидается: ERROR ... chk_ml_description'
BEGIN;
INSERT INTO maintenance_logs (car_id, manager_id, description, cost)
SELECT c.id, u.id, ' ', 100 FROM cars c, users u
WHERE c.plate_number = 'А101АА777' AND u.email = 'manager1@carsharing.local';
ROLLBACK;

\echo
\echo '=============== C. Защита истории: удаление связанных записей ==============='

\echo
\echo 'Тест 25. Удаление автомобиля, у которого есть аренды и обслуживание'
\echo 'Ожидается: ERROR ... violates foreign key constraint (fk_rentals_car или fk_ml_car)'
BEGIN;
DELETE FROM cars WHERE plate_number = 'А101АА777';
ROLLBACK;

\echo
\echo 'Тест 26. Удаление пользователя, у которого есть аренды'
\echo 'Ожидается: ERROR ... fk_rentals_user'
BEGIN;
DELETE FROM users WHERE email = 'ivan.petrov@example.com';
ROLLBACK;

\echo
\echo 'Тест 27. Удаление тарифа, на который ссылаются автомобили'
\echo 'Ожидается: ERROR ... violates foreign key constraint (fk_cars_tariff или fk_rentals_tariff)'
BEGIN;
DELETE FROM tariffs WHERE name = 'Эконом';
ROLLBACK;

\echo
\echo '=============== D. Допустимые операции (должны пройти) ==============='

\echo
\echo 'Тест 28. Удаление тарифа без связанных записей'
\echo 'Ожидается: INSERT 0 1, затем DELETE 1'
BEGIN;
INSERT INTO tariffs (name, price_per_min, waiting_per_min) VALUES ('Временный', 5, 1);
DELETE FROM tariffs WHERE name = 'Временный';
ROLLBACK;

\echo
\echo 'Тест 29. Завершение активной аренды освобождает машину для нового бронирования'
\echo 'Ожидается: UPDATE 1, затем INSERT 0 1'
BEGIN;
UPDATE rentals SET status = 'completed', ended_at = now(), total_cost = 250
WHERE status = 'active'
  AND car_id = (SELECT id FROM cars WHERE plate_number = 'Е303ЕЕ777');
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, t.price_per_min, t.waiting_per_min, 'booked'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'dmitry.ivanov@example.com' AND c.plate_number = 'Е303ЕЕ777';
ROLLBACK;

\echo
\echo 'Тест 30. Отмена бронирования и повторное бронирование той же машины'
\echo 'Ожидается: UPDATE 1, затем INSERT 0 1'
BEGIN;
UPDATE rentals SET status = 'cancelled', ended_at = now(), total_cost = 0
WHERE status = 'booked'
  AND car_id = (SELECT id FROM cars WHERE plate_number = 'К404КК777');
INSERT INTO rentals (user_id, tariff_id, car_id, price_per_min, waiting_per_min, status)
SELECT u.id, c.tariff_id, c.id, t.price_per_min, t.waiting_per_min, 'booked'
FROM users u, cars c JOIN tariffs t ON t.id = c.tariff_id
WHERE u.email = 'dmitry.ivanov@example.com' AND c.plate_number = 'К404КК777';
ROLLBACK;

\echo
\echo '=============== E. Согласованность данных (каждый запрос должен вернуть 0 строк) ==============='

\echo
\echo 'Проверка 1. Машины in_use без активной аренды'
SELECT c.plate_number, c.status
FROM cars c
WHERE c.status = 'in_use'
  AND NOT EXISTS (SELECT 1 FROM rentals r WHERE r.car_id = c.id AND r.status = 'active');

\echo
\echo 'Проверка 2. Машины booked без бронирования'
SELECT c.plate_number, c.status
FROM cars c
WHERE c.status = 'booked'
  AND NOT EXISTS (SELECT 1 FROM rentals r WHERE r.car_id = c.id AND r.status = 'booked');

\echo
\echo 'Проверка 3. Машины с активной арендой или бронированием, но статусом не in_use и не booked'
SELECT c.plate_number, c.status, r.status AS rental_status
FROM cars c
JOIN rentals r ON r.car_id = c.id AND r.status IN ('booked', 'active')
WHERE (r.status = 'active' AND c.status <> 'in_use')
   OR (r.status = 'booked' AND c.status <> 'booked');

\echo
\echo 'Проверка 4. Аренды, у которых тариф или цены отличаются от текущих значений автомобиля (на момент загрузки)'
SELECT r.id, c.plate_number, r.tariff_id AS rental_tariff, c.tariff_id AS car_tariff,
       r.price_per_min, t.price_per_min AS car_tariff_price
FROM rentals r
JOIN cars c    ON c.id = r.car_id
JOIN tariffs t ON t.id = c.tariff_id
WHERE r.tariff_id <> c.tariff_id
   OR r.price_per_min <> t.price_per_min
   OR r.waiting_per_min <> t.waiting_per_min;

\echo
\echo 'Проверка 5. Записи обслуживания, автор которых сейчас не менеджер и не администратор'
\echo '(может быть не пусто, если сотрудника позже перевели в клиенты: запись остаётся в истории)'
SELECT m.id, u.email, u.role
FROM maintenance_logs m
JOIN users u ON u.id = m.manager_id
WHERE u.role NOT IN ('manager', 'admin');

\echo
\echo 'Проверка 6. Аренды у пользователей без водительского удостоверения'
SELECT r.id, u.email
FROM rentals r
JOIN users u ON u.id = r.user_id
WHERE u.driver_license IS NULL;

\echo
\echo '=============== F. Статистика (те же расчёты, что в GET /api/stats) ==============='

\echo
\echo 'Пользователи по ролям'
SELECT role, count(*) AS users_count FROM users GROUP BY role ORDER BY role;

\echo
\echo 'Автомобили по статусам'
SELECT status, count(*) AS cars_count FROM cars GROUP BY status ORDER BY status;

\echo
\echo 'Среднее время ожидания (от бронирования до старта), минуты'
SELECT round(avg(extract(epoch FROM (started_at - booked_at)) / 60)::numeric, 1) AS avg_wait_min
FROM rentals
WHERE started_at IS NOT NULL;

\echo
\echo 'Выручка по тарифам: завершённые поездки и платные отмены бронирований'
SELECT t.name AS tariff, count(*) AS rentals_count, sum(r.total_cost) AS revenue
FROM rentals r
JOIN tariffs t ON t.id = r.tariff_id
WHERE r.status = 'completed' OR (r.status = 'cancelled' AND r.total_cost > 0)
GROUP BY t.name
ORDER BY revenue DESC;
