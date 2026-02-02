package service

import (
	"context"
	"testing"
	"time"

	"github.com/BuzzLyutic/url-shortener/internal/storage"
)

func TestShortener_Shorten(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := New(store, Config{BaseURL: "http://localhost:8080"})
	ctx := context.Background()

	t.Run("shortens valid URL", func(t *testing.T) {
		result, err := svc.Shorten(ctx, "https://example.com/page")
		if err != nil {
			t.Fatalf("Shorten() error = %v", err)
		}

		if result.ShortCode == "" {
			t.Error("Shorten() returned empty short code")
		}

		if result.OriginalURL != "https://example.com/page" {
			t.Errorf("Shorten() OriginalURL = %v, want %v", result.OriginalURL, "https://example.com/page")
		}

		if !result.IsNew {
			t.Error("Shorten() should return IsNew=true for new URL")
		}
	})

	t.Run("returns existing code for same URL", func(t *testing.T) {
		url := "https://example.com/duplicate"

		first, err := svc.Shorten(ctx, url)
		if err != nil {
			t.Fatalf("First Shorten() error = %v", err)
		}

		second, err := svc.Shorten(ctx, url)
		if err != nil {
			t.Fatalf("Second Shorten() error = %v", err)
		}

		if first.ShortCode != second.ShortCode {
			t.Errorf("Same URL got different codes: %v vs %v", first.ShortCode, second.ShortCode)
		}

		if second.IsNew {
			t.Error("Second call should return IsNew=false")
		}
	})

	t.Run("different URLs get different codes", func(t *testing.T) {
		result1, _ := svc.Shorten(ctx, "https://example.com/page1")
		result2, _ := svc.Shorten(ctx, "https://example.com/page2")

		if result1.ShortCode == result2.ShortCode {
			t.Errorf("Different URLs got same code: %v", result1.ShortCode)
		}
	})
}

func TestShortener_Shorten_Validation(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := New(store, Config{})
	ctx := context.Background()

	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			name:    "empty URL",
			url:     "",
			wantErr: ErrEmptyURL,
		},
		{
			name:    "missing scheme",
			url:     "example.com/page",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "missing host",
			url:     "https:///page",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "invalid scheme ftp",
			url:     "ftp://example.com/file",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "invalid scheme javascript",
			url:     "javascript:alert('xss')",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "valid http",
			url:     "http://example.com",
			wantErr: nil,
		},
		{
			name:    "valid https",
			url:     "https://example.com/path?query=1",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Shorten(ctx, tt.url)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Shorten() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Shorten() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestShortener_Resolve(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := New(store, Config{BaseURL: "http://localhost:8080"})
	ctx := context.Background()

	t.Run("resolves existing code", func(t *testing.T) {
		originalURL := "https://example.com/resolve-test"
		result, _ := svc.Shorten(ctx, originalURL)

		resolved, err := svc.Resolve(ctx, result.ShortCode)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}

		if resolved != originalURL {
			t.Errorf("Resolve() = %v, want %v", resolved, originalURL)
		}
	})

	t.Run("returns error for non-existent code", func(t *testing.T) {
		_, err := svc.Resolve(ctx, "nonexist12")
		if err != ErrCodeNotFound {
			t.Errorf("Resolve() error = %v, want %v", err, ErrCodeNotFound)
		}
	})

	t.Run("returns error for invalid code format", func(t *testing.T) {
		_, err := svc.Resolve(ctx, "short") // слишком короткий
		if err != ErrCodeNotFound {
			t.Errorf("Resolve() error = %v, want %v", err, ErrCodeNotFound)
		}
	})
}

func TestShortener_TTL(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := New(store, Config{
		BaseURL:    "http://localhost:8080",
		DefaultTTL: 1 * time.Hour,
	})
	ctx := context.Background()

	result, err := svc.Shorten(ctx, "https://example.com/ttl-test")
	if err != nil {
		t.Fatalf("Shorten() error = %v", err)
	}

	if result.ExpiresAt == nil {
		t.Fatal("Expected ExpiresAt to be set when DefaultTTL is configured")
	}

	expectedExpiry := time.Now().Add(1 * time.Hour)
	diff := result.ExpiresAt.Sub(expectedExpiry)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("ExpiresAt = %v, want approximately %v", result.ExpiresAt, expectedExpiry)
	}
}

func TestShortener_NoTTL(t *testing.T) {
	store := storage.NewMemoryStorage()
	svc := New(store, Config{
		BaseURL:    "http://localhost:8080",
		DefaultTTL: 0, // без TTL
	})
	ctx := context.Background()

	result, err := svc.Shorten(ctx, "https://example.com/no-ttl-test")
	if err != nil {
		t.Fatalf("Shorten() error = %v", err)
	}

	if result.ExpiresAt != nil {
		t.Errorf("Expected ExpiresAt to be nil when DefaultTTL is 0, got %v", result.ExpiresAt)
	}
}

func TestShortener_BuildShortURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		code    string
		want    string
	}{
		{
			name:    "with base URL",
			baseURL: "http://localhost:8080",
			code:    "aB3_xY9z12",
			want:    "http://localhost:8080/aB3_xY9z12",
		},
		{
			name:    "empty base URL",
			baseURL: "",
			code:    "aB3_xY9z12",
			want:    "aB3_xY9z12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(nil, Config{BaseURL: tt.baseURL})
			got := svc.buildShortURL(tt.code)
			if got != tt.want {
				t.Errorf("buildShortURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// mockStorage реализует storage.Storage для тестирования обработки коллизий.
type mockStorage struct {
	storage.Storage
	saveCalls      int
	failUntilAttempt int
}

func (m *mockStorage) Save(url storage.URL) error {
	m.saveCalls++
	if m.saveCalls <= m.failUntilAttempt {
		return storage.ErrAlreadyExists
	}
	return nil
}

func (m *mockStorage) GetByOriginalURL(originalURL string) (*storage.URL, error) {
	return nil, storage.ErrNotFound
}

func (m *mockStorage) GetByCode(code string) (*storage.URL, error) {
	return nil, storage.ErrNotFound
}

func TestShortener_CollisionHandling(t *testing.T) {
	t.Run("retries on collision", func(t *testing.T) {
		mock := &mockStorage{failUntilAttempt: 2}
		svc := New(mock, Config{})
		ctx := context.Background()

		_, err := svc.Shorten(ctx, "https://example.com/collision")
		if err != nil {
			t.Fatalf("Shorten() error = %v", err)
		}

		if mock.saveCalls != 3 {
			t.Errorf("Expected 3 save attempts, got %d", mock.saveCalls)
		}
	})

	t.Run("fails after max attempts", func(t *testing.T) {
		mock := &mockStorage{failUntilAttempt: maxAttempts + 1}
		svc := New(mock, Config{})
		ctx := context.Background()

		_, err := svc.Shorten(ctx, "https://example.com/too-many-collisions")
		if err != ErrTooManyCollisions {
			t.Errorf("Shorten() error = %v, want %v", err, ErrTooManyCollisions)
		}

		if mock.saveCalls != maxAttempts {
			t.Errorf("Expected %d save attempts, got %d", maxAttempts, mock.saveCalls)
		}
	})
}

// Бенчмарки

func BenchmarkShortener_Shorten(b *testing.B) {
	store := storage.NewMemoryStorage()
	svc := New(store, Config{BaseURL: "http://localhost:8080"})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := "https://example.com/page/" + string(rune('A'+i%26))
		_, _ = svc.Shorten(ctx, url)
	}
}

func BenchmarkShortener_Resolve(b *testing.B) {
	store := storage.NewMemoryStorage()
	svc := New(store, Config{BaseURL: "http://localhost:8080"})
	ctx := context.Background()

	result, _ := svc.Shorten(ctx, "https://example.com/bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.Resolve(ctx, result.ShortCode)
	}
}
