package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQL реализация хранилища
type PostgresStorage struct {
	db *sql.DB
}

// Конфигурация подключения для PostgreSQL
type PostgresConfig struct {
	DSN             string        // Строка подключения
	MaxOpenConns    int           // Макс. открытых соединений
	MaxIdleConns    int           // Макс. незанятых соединений
	ConnMaxLifetime time.Duration // Макс. время жизни соединения
	ConnMaxIdleTime time.Duration // Макс. время жизни незанятого соединения
}

// Конфиг Postgres по умолчанию
func DefaultPostgresConfig(dsn string) PostgresConfig {
	return PostgresConfig{
		DSN:             dsn,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

func NewPostgresStorage(cfg PostgresConfig) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Кофиг пулов соединений
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Подтверждение соединения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

// Save сохраняет новое URL отображение
func (s *PostgresStorage) Save(url URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO urls (short_code, original_url, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (short_code) DO NOTHING
	`

	result, err := s.db.ExecContext(ctx, query,
		url.ShortCode,
		url.OriginalURL,
		url.CreatedAt,
		url.ExpiresAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("inserting URL: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Проверить, это тот же самый URL (идемпотентность) или другой (коллизия)
		existing, err := s.GetByCode(url.ShortCode)
		if err != nil {
			return fmt.Errorf("checking existing code: %w", err)
		}
		if existing.OriginalURL != url.OriginalURL {
			return ErrAlreadyExists
		}
	}

	return nil
}

// GetByCode возвращает URL по короткому коду
func (s *PostgresStorage) GetByCode(code string) (*URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT short_code, original_url, created_at, expires_at
		FROM urls
		WHERE short_code = $1
	`

	var url URL
	err := s.db.QueryRowContext(ctx, query, code).Scan(
		&url.ShortCode,
		&url.OriginalURL,
		&url.CreatedAt,
		&url.ExpiresAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying URL by code: %w", err)
	}

	if url.IsExpired() {
		return nil, ErrExpired
	}

	return &url, nil
}

// Возвращает укороченную ссылку по оригинальному URL
func (s *PostgresStorage) GetByOriginalURL(originalURL string) (*URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT short_code, original_url, created_at, expires_at
		FROM urls
		WHERE original_url = $1
	`

	var url URL
	err := s.db.QueryRowContext(ctx, query, originalURL).Scan(
		&url.ShortCode,
		&url.OriginalURL,
		&url.CreatedAt,
		&url.ExpiresAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying URL by original: %w", err)
	}

	if url.IsExpired() {
		return nil, ErrExpired
	}

	return &url, nil
}

// Close закрывает соединение с БД
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// Ping проверяет соединение с БД
func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// isUniqueViolation проверяет наличие нарушения ограничения уникальности
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL 23505 код ошибки = unique_violation
	return !errors.Is(err, sql.ErrNoRows) &&
		(errorContains(err, "23505") || errorContains(err, "unique constraint"))
}

func errorContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	return !errors.Is(err, sql.ErrNoRows) && contains(err.Error(), substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
