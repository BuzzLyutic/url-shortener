CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(10) NOT NULL UNIQUE,
    original_url TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

-- Индекс для быстрого поиска по короткому коду
CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);

-- Индекс для проверки существующих оригинальных URL
CREATE INDEX IF NOT EXISTS idx_urls_original_url ON urls(original_url);

-- Индекс для очистки устаревших URL
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at) 
    WHERE expires_at IS NOT NULL;
