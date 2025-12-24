-- +goose Up

-- Создаем таблицу пользователей
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    phone TEXT UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создаем таблицу призов
CREATE TABLE IF NOT EXISTS prizes (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    prize TEXT NOT NULL,
    telegram_id BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    used_at TIMESTAMP
);

-- Индексы для производительности
CREATE INDEX IF NOT EXISTS idx_prizes_telegram_id ON prizes(telegram_id);
CREATE INDEX IF NOT EXISTS idx_prizes_code ON prizes(code);

-- +goose Down

DROP INDEX IF EXISTS idx_prizes_code;
DROP INDEX IF EXISTS idx_prizes_telegram_id;
DROP TABLE IF EXISTS prizes;
DROP TABLE IF EXISTS users;
