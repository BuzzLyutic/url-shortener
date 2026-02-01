package storage

import (
	"sync"
	"testing"
	"time"
)

func TestMemoryStorage_Save(t *testing.T) {
	s := NewMemoryStorage()

	url := URL{
		ShortCode:   "aB3_xY9z12",
		OriginalURL: "https://example.com",
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
		ShortCode:   "aB3_xY9z12",
		OriginalURL: "https://different.com",
		CreatedAt:   time.Now(),
	}
	err = s.Save(url2)
	if err != ErrAlreadyExists {
		t.Errorf("Save() error = %v, want %v", err, ErrAlreadyExists)
	}
}

func TestMemoryStorage_GetByCode(t *testing.T) {
	s := NewMemoryStorage()

	url := URL{
		ShortCode:   "aB3_xY9z12",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
	}
	_ = s.Save(url)

	// Должен найти существующий URL
	got, err := s.GetByCode("aB3_xY9z12")
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if got.OriginalURL != url.OriginalURL {
		t.Errorf("GetByCode() OriginalURL = %v, want %v", got.OriginalURL, url.OriginalURL)
	}

	// Должен вернуть ErrNotFound для несуществующего кода
	_, err = s.GetByCode("nonexistent")
	if err != ErrNotFound {
		t.Errorf("GetByCode() error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemoryStorage_GetByOriginalURL(t *testing.T) {
	s := NewMemoryStorage()

	url := URL{
		ShortCode:   "aB3_xY9z12",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
	}
	_ = s.Save(url)

	got, err := s.GetByOriginalURL("https://example.com")
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

func TestMemoryStorage_Expiration(t *testing.T) {
	s := NewMemoryStorage()

	pastTime := time.Now().Add(-1 * time.Hour)
	url := URL{
		ShortCode:   "expired123",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
		ExpiresAt:   &pastTime,
	}
	_ = s.Save(url)

	_, err := s.GetByCode("expired123")
	if err != ErrExpired {
		t.Errorf("GetByCode() error = %v, want %v", err, ErrExpired)
	}

	_, err = s.GetByOriginalURL("https://example.com")
	if err != ErrExpired {
		t.Errorf("GetByOriginalURL() error = %v, want %v", err, ErrExpired)
	}
}

func TestMemoryStorage_NotExpired(t *testing.T) {
	s := NewMemoryStorage()

	futureTime := time.Now().Add(1 * time.Hour)
	url := URL{
		ShortCode:   "future1234",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
		ExpiresAt:   &futureTime,
	}
	_ = s.Save(url)

	got, err := s.GetByCode("future1234")
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if got.OriginalURL != url.OriginalURL {
		t.Errorf("GetByCode() OriginalURL = %v, want %v", got.OriginalURL, url.OriginalURL)
	}
}

func TestMemoryStorage_NoExpiration(t *testing.T) {
	s := NewMemoryStorage()

	url := URL{
		ShortCode:   "noexpire12",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
		ExpiresAt:   nil,
	}
	_ = s.Save(url)

	got, err := s.GetByCode("noexpire12")
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if got.OriginalURL != url.OriginalURL {
		t.Errorf("GetByCode() OriginalURL = %v, want %v", got.OriginalURL, url.OriginalURL)
	}
}

func TestMemoryStorage_Concurrent(t *testing.T) {
	s := NewMemoryStorage()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				url := URL{
					ShortCode:   "code" + string(rune('A'+id%26)) + string(rune('0'+j%10)),
					OriginalURL: "https://example.com/" + string(rune('A'+id%26)) + string(rune('0'+j%10)),
					CreatedAt:   time.Now(),
				}
				_ = s.Save(url)
				_, _ = s.GetByCode(url.ShortCode)
				_, _ = s.GetByOriginalURL(url.OriginalURL)
			}
		}(i)
	}

	wg.Wait()
}

func TestURL_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "nil expiration",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "future expiration",
			expiresAt: func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
			want:      false,
		},
		{
			name:      "past expiration",
			expiresAt: func() *time.Time { t := time.Now().Add(-time.Hour); return &t }(),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &URL{ExpiresAt: tt.expiresAt}
			if got := u.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Бенчмарки

func BenchmarkMemoryStorage_Save(b *testing.B) {
	s := NewMemoryStorage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := URL{
			ShortCode:   "code" + string(rune('A'+i%26)),
			OriginalURL: "https://example.com/" + string(rune('A'+i%26)),
			CreatedAt:   time.Now(),
		}
		_ = s.Save(url)
	}
}

func BenchmarkMemoryStorage_GetByCode(b *testing.B) {
	s := NewMemoryStorage()

	url := URL{
		ShortCode:   "aB3_xY9z12",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
	}
	_ = s.Save(url)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.GetByCode("aB3_xY9z12")
	}
}

func BenchmarkMemoryStorage_GetByOriginalURL(b *testing.B) {
	s := NewMemoryStorage()

	url := URL{
		ShortCode:   "aB3_xY9z12",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
	}
	_ = s.Save(url)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.GetByOriginalURL("https://example.com")
	}
}

func BenchmarkMemoryStorage_Concurrent(b *testing.B) {
	s := NewMemoryStorage()

	for i := 0; i < 1000; i++ {
		url := URL{
			ShortCode:   "code" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
			OriginalURL: "https://example.com/" + string(rune(i)),
			CreatedAt:   time.Now(),
		}
		_ = s.Save(url)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			code := "code" + string(rune('A'+i%26)) + string(rune('0'+i%10))
			_, _ = s.GetByCode(code)
			i++
		}
	})
}
