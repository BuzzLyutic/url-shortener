// Пакет handler предоставляет хэндлеры для API укорачивания ссылок
package handler

import "time"

// Тело запроса
type ShortenRequest struct {
	URL string `json:"url"`
}

// Тело ответа
type ShortenResponse struct {
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Ответ ошибки
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
