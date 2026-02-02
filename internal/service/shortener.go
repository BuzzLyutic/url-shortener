// Пакет service реализует бизнес-логику для укорачивания ссылок.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/BuzzLyutic/url-shortener/internal/shortcode"
	"github.com/BuzzLyutic/url-shortener/internal/storage"
)

// Кастомные ошибки, возвращаемые сервисом
var (
	ErrInvalidURL        = errors.New("invalid URL")
	ErrEmptyURL          = errors.New("URL cannot be empty")
	ErrCodeNotFound      = errors.New("short code not found")
	ErrTooManyCollisions = errors.New("failed to generate unique code after max attempts")
)

const (
	maxAttempts = 10 // Максимальное кол-во попыток разрешения коллизий
)

// Config содержит конфиг сервиса
type Config struct {
	BaseURL    string        // Базовый URL для коротких ссылок
	DefaultTTL time.Duration // TTL для ссылок по умолчанию
}

// Shortener предоставляет операции для укорачивания ссылок
type Shortener struct {
	storage storage.Storage
	config  Config
}

// New создает новый сервис Shortener
func New(store storage.Storage, config Config) *Shortener {
	return &Shortener{
		storage: store,
		config:  config,
	}
}

// ShortenResult содержит результат укорачивания ссылок
type ShortenResult struct {
	ShortCode   string
	ShortURL    string
	OriginalURL string
	ExpiresAt   *time.Time
	IsNew       bool // true если новый короткий код создан
}

// Shorten создает укороченную ссылку по оригинальному URL
// Если URL уже был укорочен ранее, возвращается существующий код
func (s *Shortener) Shorten(ctx context.Context, originalURL string) (*ShortenResult, error) {
	// Валидация URL
	if err := s.validateURL(originalURL); err != nil {
		return nil, err
	}

	// Проверить существование URL
	existing, err := s.storage.GetByOriginalURL(originalURL)
	if err == nil {
		// URL уже сокращен
		return &ShortenResult{
			ShortCode:   existing.ShortCode,
			ShortURL:    s.buildShortURL(existing.ShortCode),
			OriginalURL: existing.OriginalURL,
			ExpiresAt:   existing.ExpiresAt,
			IsNew:       false,
		}, nil
	}

	if !errors.Is(err, storage.ErrNotFound) && !errors.Is(err, storage.ErrExpired) {
		return nil, fmt.Errorf("checking existing URL: %w", err)
	}

	// Сгенерировать новый короткий код с обработкой коллизий
	for attempt := 0; attempt < maxAttempts; attempt++ {
		code := shortcode.Generate(originalURL, attempt)

		// Рассчитать срок истечения
		var expiresAt *time.Time
		if s.config.DefaultTTL > 0 {
			t := time.Now().Add(s.config.DefaultTTL)
			expiresAt = &t
		}

		urlRecord := storage.URL{
			ShortCode:   code,
			OriginalURL: originalURL,
			CreatedAt:   time.Now(),
			ExpiresAt:   expiresAt,
		}

		err := s.storage.Save(urlRecord)
		if err == nil {
			// Успешное сохранение
			return &ShortenResult{
				ShortCode:   code,
				ShortURL:    s.buildShortURL(code),
				OriginalURL: originalURL,
				ExpiresAt:   expiresAt,
				IsNew:       true,
			}, nil
		}

		if errors.Is(err, storage.ErrAlreadyExists) {
			// Коллизия
			continue
		}
		return nil, fmt.Errorf("saving URL: %w", err)
	}

	return nil, ErrTooManyCollisions
}

// Resolve возвращает оригинальный URL по короткой ссылке
func (s *Shortener) Resolve(ctx context.Context, code string) (string, error) {
	if !shortcode.IsValid(code) {
		return "", ErrCodeNotFound
	}

	urlRecord, err := s.storage.GetByCode(code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrExpired) {
			return "", ErrCodeNotFound
		}
		return "", fmt.Errorf("getting URL: %w", err)
	}

	return urlRecord.OriginalURL, nil
}

// validateURL проверяет валидность URL
func (s *Shortener) validateURL(rawURL string) error {
	if rawURL == "" {
		return ErrEmptyURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}

	return nil
}

// buildShortURL собирает полную укороченную строку
func (s *Shortener) buildShortURL(code string) string {
	if s.config.BaseURL == "" {
		return code
	}
	return s.config.BaseURL + "/" + code
}
