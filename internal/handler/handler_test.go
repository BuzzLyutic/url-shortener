package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/BuzzLyutic/url-shortener/internal/service"
	"github.com/BuzzLyutic/url-shortener/internal/storage"
)

func setupTestHandler() (*Handler, *http.ServeMux) {
	store := storage.NewMemoryStorage()
	svc := service.New(store, service.Config{
		BaseURL: "http://localhost:8080",
	})
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	h := New(svc, logger)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	return h, mux
}

func TestHandler_Shorten(t *testing.T) {
	_, mux := setupTestHandler()

	t.Run("successful shorten", func(t *testing.T) {
		body := `{"url": "https://example.com/test"}`
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusCreated)
		}

		var resp ShortenResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.ShortURL == "" {
			t.Error("ShortURL is empty")
		}

		if resp.OriginalURL != "https://example.com/test" {
			t.Errorf("OriginalURL = %s, want %s", resp.OriginalURL, "https://example.com/test")
		}
	})

	t.Run("duplicate URL returns 200", func(t *testing.T) {
		body := `{"url": "https://example.com/duplicate"}`

		// Первый запрос
		req1 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
		rec1 := httptest.NewRecorder()
		mux.ServeHTTP(rec1, req1)

		// Второй запрос
		req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
		rec2 := httptest.NewRecorder()
		mux.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("Second request status = %d, want %d", rec2.Code, http.StatusOK)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("{invalid}"))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty URL", func(t *testing.T) {
		body := `{"url": ""}`
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var resp ErrorResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Error != "empty_url" {
			t.Errorf("Error = %s, want %s", resp.Error, "empty_url")
		}
	})

	t.Run("invalid URL format", func(t *testing.T) {
		body := `{"url": "not-a-url"}`
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestHandler_Redirect(t *testing.T) {
	_, mux := setupTestHandler()

	t.Run("successful redirect", func(t *testing.T) {
		// Сначала создается короткий URL
		body := `{"url": "https://example.com/redirect-test"}`
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		var resp ShortenResponse
		json.NewDecoder(rec.Body).Decode(&resp)

		// Из короткого URL нужно достать код
		parts := strings.Split(resp.ShortURL, "/")
		code := parts[len(parts)-1]

		// А теперь тестируется редирект
		req = httptest.NewRequest(http.MethodGet, "/"+code, nil)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusMovedPermanently)
		}

		location := rec.Header().Get("Location")
		if location != "https://example.com/redirect-test" {
			t.Errorf("Location = %s, want %s", location, "https://example.com/redirect-test")
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexist12", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid code format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/short", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestHandler_Health(t *testing.T) {
	_, mux := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("Status = %s, want %s", resp["status"], "ok")
	}
}

func TestMiddleware_Logging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Logging(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "http request") {
		t.Error("Log message not found")
	}
	if !strings.Contains(buf.String(), "GET") {
		t.Error("Method not logged")
	}
}

func TestMiddleware_Recovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrapped := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Не должен запаниковать
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// Бенчмарки

func BenchmarkHandler_Shorten(b *testing.B) {
	_, mux := setupTestHandler()
	body := []byte(`{"url": "https://example.com/bench"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkHandler_Redirect(b *testing.B) {
	_, mux := setupTestHandler()

	body := []byte(`{"url": "https://example.com/bench-redirect"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp ShortenResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	parts := strings.Split(resp.ShortURL, "/")
	code := parts[len(parts)-1]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}
