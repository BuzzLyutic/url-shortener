// Пакет storage интерфейсы хранилищ url.
package storage

import (
	"errors"
	"time"
)

// Кастомные ошибки для реализаций хранилищ
var (
	ErrNotFound      = errors.New("url not found")
	ErrAlreadyExists = errors.New("short code already exists")
	ErrExpired       = errors.New("url has expired")
)

// URL представляет собой сохраненное отображение URL
type URL struct {
	ShortCode   string
	OriginalURL string
	CreatedAt   time.Time
	ExpiresAt   *time.Time // nil означает отсутствие срока истечения
}

// IsExpired проверяет, истек ли срок жизни URL
func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.ExpiresAt)
}

// Storage определяет интерфейс хранилища URL
type Storage interface {
	Save(url URL) error                                // Save хранит новое отображение URL.
	GetByCode(code string) (*URL, error)               // GetByCode возвращает URL по короткому коду.
	GetByOriginalURL(originalURL string) (*URL, error) // GetByOriginalURL возвращает новый URL по оригинальной ссылке.
	Close() error                                      // Close закрывает хранилище и освобождает ресурсы.
}
