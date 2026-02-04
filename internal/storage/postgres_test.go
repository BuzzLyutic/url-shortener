package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

// getTestDSN возвращает DSN БД для тестов
func getTestDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	return dsn
}

// Пропуск тестов при недоступности БД
func skipIfNoDatabase(t *testing.T) *PostgresStorage {
	t.Helper()

	dsn := getTestDSN()
	if dsn == "" {
		t.Skip("Skipping PostgreSQL test: DATABASE_URL or TEST_DATABASE_URL not set")
	}

	cfg := DefaultPostgresConfig(dsn)
	storage, err := NewPostgresStorage(cfg)
	if err != nil {
		t.Skipf("Skipping PostgreSQL test: cannot connect to database: %v", err)
	}

	// Очистка таблиц перед тестированием
	ctx := context.Background()
	_, _ = storage.db.ExecContext(ctx, "DELETE FROM urls")

	return storage
}

func TestPostgresStorage_Save(t *testing.T) {
	s := skipIfNoDatabase(t)
	defer s.Close()

	url := URL{
		ShortCode:   "testcode12",
		OriginalURL: "https://example.com/postgres-test",
		CreatedAt:   time.Now(),
	}

	err := s.Save(url)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	err = s.Save(url)
	if err != nil {
		t.Fatalf("Save() idempotent error = %v", err)
	}

	url2 := URL{
		ShortCode:   "testcode12",
		OriginalURL: "https://different.com",
		CreatedAt:   time.Now(),
	}
	err = s.Save(url2)
	if err != ErrAlreadyExists {
		t.Errorf("Save() error = %v, want %v", err, ErrAlreadyExists)
	}
}

func TestPostgresStorage_GetByCode(t *testing.T) {
	s := skipIfNoDatabase(t)
	defer s.Close()

	url := URL{
		ShortCode:   "getcode123",
		OriginalURL: "https://example.com/get-by-code",
		CreatedAt:   time.Now(),
	}
	_ = s.Save(url)

	got, err := s.GetByCode("getcode123")
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if got.OriginalURL != url.OriginalURL {
		t.Errorf("GetByCode() OriginalURL = %v, want %v", got.OriginalURL, url.OriginalURL)
	}

	_, err = s.GetByCode("nonexist12")
	if err != ErrNotFound {
		t.Errorf("GetByCode() error = %v, want %v", err, ErrNotFound)
	}
}

func TestPostgresStorage_GetByOriginalURL(t *testing.T) {
	s := skipIfNoDatabase(t)
	defer s.Close()

	url := URL{
		ShortCode:   "origurl123",
		OriginalURL: "https://example.com/get-by-original",
		CreatedAt:   time.Now(),
	}
	_ = s.Save(url)

	got, err := s.GetByOriginalURL("https://example.com/get-by-original")
	if err != nil {
		t.Fatalf("GetByOriginalURL() error = %v", err)
	}
	if got.ShortCode != url.ShortCode {
		t.Errorf("GetByOriginalURL() ShortCode = %v, want %v", got.ShortCode, url.ShortCode)
	}

	_, err = s.GetByOriginalURL("https://nonexistent.com")
	if err != ErrNotFound {
		t.Errorf("GetByOriginalURL() error = %v, want %v", err, ErrNotFound)
	}
}

func TestPostgresStorage_Expiration(t *testing.T) {
	s := skipIfNoDatabase(t)
	defer s.Close()

	// Создать уже устаревший URL
	pastTime := time.Now().Add(-1 * time.Hour)
	url := URL{
		ShortCode:   "expired123",
		OriginalURL: "https://example.com/expired-pg",
		CreatedAt:   time.Now(),
		ExpiresAt:   &pastTime,
	}
	_ = s.Save(url)

	_, err := s.GetByCode("expired123")
	if err != ErrExpired {
		t.Errorf("GetByCode() error = %v, want %v", err, ErrExpired)
	}

	_, err = s.GetByOriginalURL("https://example.com/expired-pg")
	if err != ErrExpired {
		t.Errorf("GetByOriginalURL() error = %v, want %v", err, ErrExpired)
	}
}

func TestPostgresStorage_Ping(t *testing.T) {
	s := skipIfNoDatabase(t)
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.Ping(ctx)
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

// Бенчмарки

func BenchmarkPostgresStorage_Save(b *testing.B) {
	dsn := getTestDSN()
	if dsn == "" {
		b.Skip("Skipping: DATABASE_URL not set")
	}

	cfg := DefaultPostgresConfig(dsn)
	s, err := NewPostgresStorage(cfg)
	if err != nil {
		b.Skipf("Cannot connect: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_, _ = s.db.ExecContext(ctx, "DELETE FROM urls")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := URL{
			ShortCode:   "bench" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			OriginalURL: "https://example.com/bench/" + string(rune(i)),
			CreatedAt:   time.Now(),
		}
		_ = s.Save(url)
	}
}

func BenchmarkPostgresStorage_GetByCode(b *testing.B) {
	dsn := getTestDSN()
	if dsn == "" {
		b.Skip("Skipping: DATABASE_URL not set")
	}

	cfg := DefaultPostgresConfig(dsn)
	s, err := NewPostgresStorage(cfg)
	if err != nil {
		b.Skipf("Cannot connect: %v", err)
	}
	defer s.Close()

	url := URL{
		ShortCode:   "benchget12",
		OriginalURL: "https://example.com/bench-get",
		CreatedAt:   time.Now(),
	}
	_ = s.Save(url)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.GetByCode("benchget12")
	}
}
