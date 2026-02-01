package storage

import (
	"sync"
)

// MemoryStorage реализация хранилища в памяти
type MemoryStorage struct {
	mu            sync.RWMutex
	byCode        map[string]*URL
	byOriginalURL map[string]string // Оригинальный URL -> укороченный код
}

// NewMemoryStorage создает новое хранилище в памяти
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		byCode:        make(map[string]*URL),
		byOriginalURL: make(map[string]string),
	}
}

// Save хранит новое отображение URL
func (s *MemoryStorage) Save(url URL) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверка существования кода
	if existing, ok := s.byCode[url.ShortCode]; ok {
		if existing.OriginalURL == url.OriginalURL {
			// один URL должен соответствовать одному и тому же коду
			return nil
		}
		return ErrAlreadyExists
	}

	// Сохранить URL
	urlCopy := url // создать копию, чтобы избежать внешних изменений
	s.byCode[url.ShortCode] = &urlCopy
	s.byOriginalURL[url.OriginalURL] = url.ShortCode

	return nil
}

// GetByCode возвращает URL по короткому коду
func (s *MemoryStorage) GetByCode(code string) (*URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.byCode[code]
	if !ok {
		return nil, ErrNotFound
	}

	if url.IsExpired() {
		return nil, ErrExpired
	}

	return url, nil
}

// GetByOriginalURL возвращает укороченную ссылку по оригинальному URL
func (s *MemoryStorage) GetByOriginalURL(originalURL string) (*URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	code, ok := s.byOriginalURL[originalURL]
	if !ok {
		return nil, ErrNotFound
	}

	url := s.byCode[code]
	if url.IsExpired() {
		return nil, ErrExpired
	}

	return url, nil
}

// Close закрывает хранилище. Для хранения данных в памяти это не требуется
func (s *MemoryStorage) Close() error {
	return nil
}

// Len возвращает кол-во сохраненных URLs (для тестов)
func (s *MemoryStorage) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byCode)
}
