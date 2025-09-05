-- Таблица заказов
CREATE TABLE IF NOT EXISTS orders (
                        order_uid TEXT PRIMARY KEY,
                        track_number TEXT NOT NULL,
                        entry TEXT,
                        locale TEXT,
                        internal_signature TEXT,
                        customer_id TEXT,
                        delivery_service TEXT,
                        shardkey TEXT,
                        sm_id INT,
                        date_created TIMESTAMP,
                        oof_shard TEXT
);

-- Таблица доставки
CREATE TABLE IF NOT EXISTS delivery (
                          order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
                          name TEXT,
                          phone TEXT,
                          zip TEXT,
                          city TEXT,
                          address TEXT,
                          region TEXT,
                          email TEXT
);

-- Таблица оплаты
CREATE TABLE IF NOT EXISTS payment(
                         order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
                         transaction TEXT,
                         request_id TEXT,
                         currency TEXT,
                         provider TEXT,
                         amount INT,
                         payment_dt BIGINT,
                         bank TEXT,
                         delivery_cost INT,
                         goods_total INT,
                         custom_fee INT
);

-- Таблица товаров
CREATE TABLE IF NOT EXISTS items (
                       order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
                       chrt_id BIGINT,
                       track_number TEXT,
                       price INT,
                       rid TEXT,
                       name TEXT,
                       sale INT,
                       size TEXT,
                       total_price INT,
                       nm_id BIGINT,
                       brand TEXT,
                       status INT
);

-- Создание пользователя
DO $$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'wb_user') THEN
            CREATE USER wb_user WITH PASSWORD 'password';
        END IF;
    END
$$;

-- Назначаем права пользователю на уже существующую базу
GRANT ALL PRIVILEGES ON DATABASE postgres TO wb_user;

-- Подключаем схему public
GRANT ALL PRIVILEGES ON SCHEMA public TO wb_user;

-- Доступ ко всем существующим таблицам
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO wb_user;

-- Доступ к последовательностям (например, для SERIAL/IDENTITY)
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO wb_user;

-- Автоматическая выдача прав на будущие объекты
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO wb_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO wb_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON FUNCTIONS TO wb_user;
