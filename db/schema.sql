CREATE EXTENSION IF NOT EXISTS citext;

-- -------------------------------------------------------------
-- Повторный запуск на учебной базе: раскомментируйте блок ниже.
-- ВНИМАНИЕ: удаляет все данные.
-- -------------------------------------------------------------
-- DROP TABLE IF EXISTS maintenance_logs CASCADE;
-- DROP TABLE IF EXISTS rentals          CASCADE;
-- DROP TABLE IF EXISTS cars             CASCADE;
-- DROP TABLE IF EXISTS users            CASCADE;
-- DROP TABLE IF EXISTS tariffs          CASCADE;


-- 1. tariffs — тарифы

CREATE TABLE tariffs (
    id              SERIAL        PRIMARY KEY,
    name            VARCHAR(50)   NOT NULL,
    price_per_min   NUMERIC(8,2)  NOT NULL,
    waiting_per_min NUMERIC(8,2)  NOT NULL,

    CONSTRAINT uq_tariffs_name          UNIQUE (name),
    CONSTRAINT chk_tariffs_price        CHECK (price_per_min   >= 0),
    CONSTRAINT chk_tariffs_waiting      CHECK (waiting_per_min >= 0)
);


-- 2. users — пользователи (клиенты, менеджеры, администраторы)

CREATE TABLE users (
    id             BIGSERIAL     PRIMARY KEY,
    email          CITEXT        NOT NULL,
    password_hash  VARCHAR(255)  NOT NULL,
    full_name      VARCHAR(100)  NOT NULL,
    driver_license VARCHAR(20),             
    role           VARCHAR(20)   NOT NULL DEFAULT 'client',
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_users_email           UNIQUE (email),
    CONSTRAINT uq_users_driver_license  UNIQUE (driver_license),
    CONSTRAINT chk_users_role           CHECK (role IN ('client', 'manager', 'admin'))
);


-- 3. cars — автомобили

CREATE TABLE cars (
    id           BIGSERIAL         PRIMARY KEY,
    tariff_id    INT               NOT NULL,
    brand_model  VARCHAR(100)      NOT NULL,
    plate_number VARCHAR(15)       NOT NULL,
    fuel_level   INT               NOT NULL,
    latitude     DOUBLE PRECISION  NOT NULL,
    longitude    DOUBLE PRECISION  NOT NULL,
    status       VARCHAR(20)       NOT NULL DEFAULT 'available',

    CONSTRAINT fk_cars_tariff
        FOREIGN KEY (tariff_id) REFERENCES tariffs (id) ON DELETE RESTRICT,
    CONSTRAINT uq_cars_plate_number     UNIQUE (plate_number),
    CONSTRAINT chk_cars_fuel_level      CHECK (fuel_level BETWEEN 0 AND 100),
    CONSTRAINT chk_cars_latitude        CHECK (latitude   BETWEEN  -90 AND  90),
    CONSTRAINT chk_cars_longitude       CHECK (longitude  BETWEEN -180 AND 180),
    CONSTRAINT chk_cars_status          CHECK (status IN
        ('available', 'booked', 'in_use', 'maintenance', 'decommissioned'))
);


-- 4. rentals — аренды (бронирования и поездки)

CREATE TABLE rentals (
    id              BIGSERIAL      PRIMARY KEY,
    user_id         BIGINT         NOT NULL,
    tariff_id       INT            NOT NULL,   
    car_id          BIGINT         NOT NULL,
    booked_at       TIMESTAMPTZ    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,
    price_per_min   NUMERIC(8,2)   NOT NULL,   
    waiting_per_min NUMERIC(8,2)   NOT NULL,   
    total_cost      NUMERIC(10,2),             
    status          VARCHAR(20)    NOT NULL DEFAULT 'booked',

    CONSTRAINT fk_rentals_user
        FOREIGN KEY (user_id)   REFERENCES users   (id) ON DELETE RESTRICT,
    CONSTRAINT fk_rentals_tariff
        FOREIGN KEY (tariff_id) REFERENCES tariffs (id) ON DELETE RESTRICT,
    CONSTRAINT fk_rentals_car
        FOREIGN KEY (car_id)    REFERENCES cars    (id) ON DELETE RESTRICT,

    CONSTRAINT chk_rentals_status
        CHECK (status IN ('booked', 'active', 'completed', 'cancelled')),

    -- Хронология: booked_at <= started_at <= ended_at
    CONSTRAINT chk_rentals_started_after_booked
        CHECK (started_at IS NULL OR started_at >= booked_at),
        
    CONSTRAINT chk_rentals_ended_after_started
        CHECK (ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at),

    -- Согласованность статуса и дат
    CONSTRAINT chk_rentals_status_dates CHECK (
           (status IN ('booked', 'cancelled')
                AND started_at IS NULL AND ended_at IS NULL)
        OR (status = 'active'
                AND started_at IS NOT NULL AND ended_at IS NULL)
        OR (status = 'completed'
                AND started_at IS NOT NULL AND ended_at IS NOT NULL
                AND total_cost IS NOT NULL)
    ),

    CONSTRAINT chk_rentals_price        CHECK (price_per_min   >= 0),
    CONSTRAINT chk_rentals_waiting      CHECK (waiting_per_min >= 0),
    CONSTRAINT chk_rentals_total_cost   CHECK (total_cost IS NULL OR total_cost >= 0)
);

-- Одна машина — одна активная аренда (бронирование или поездка):
-- второе бронирование или аренда той же машины будет отклонено базой данных.
CREATE UNIQUE INDEX uq_rentals_one_active_per_car
    ON rentals (car_id)
    WHERE status IN ('booked', 'active');


-- 5. maintenance_logs — журнал технического обслуживания

CREATE TABLE maintenance_logs (
    id          BIGSERIAL      PRIMARY KEY,
    car_id      BIGINT         NOT NULL,
    manager_id  BIGINT         NOT NULL,   -- роль manager проверяется в приложении
    description VARCHAR(255)   NOT NULL,
    cost        NUMERIC(10,2)  NOT NULL,
    logged_at   TIMESTAMPTZ    NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_ml_car
        FOREIGN KEY (car_id)     REFERENCES cars  (id) ON DELETE RESTRICT,
    CONSTRAINT fk_ml_manager
        FOREIGN KEY (manager_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT chk_ml_cost      CHECK (cost >= 0)
);

-- Индексы для внешних ключей и частых запросов

CREATE INDEX idx_cars_tariff_id      ON cars             (tariff_id);
CREATE INDEX idx_cars_status         ON cars             (status);       -- каталог: фильтр по статусу
CREATE INDEX idx_rentals_user_id     ON rentals          (user_id);      -- история поездок клиента
CREATE INDEX idx_rentals_car_id      ON rentals          (car_id);       -- история аренд автомобиля
CREATE INDEX idx_rentals_tariff_id   ON rentals          (tariff_id);
CREATE INDEX idx_ml_car_id           ON maintenance_logs (car_id);       -- история обслуживания машины
CREATE INDEX idx_ml_manager_id       ON maintenance_logs (manager_id);


-- Комментарии к таблицам (видны в psql через \dt+ и в pgAdmin)

COMMENT ON TABLE tariffs          IS 'Тарифы: цена минуты аренды и минуты ожидания';
COMMENT ON TABLE users            IS 'Пользователи системы: client, manager, admin';
COMMENT ON TABLE cars             IS 'Автопарк; координаты используются картой Leaflet';
COMMENT ON TABLE rentals          IS 'Аренды: бронирования и поездки; цены тарифа копируются при создании';
COMMENT ON TABLE maintenance_logs IS 'Журнал технического обслуживания автомобилей';
