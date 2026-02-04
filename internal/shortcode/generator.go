// Пакет shortcode предоставляет функциональность генерации коротких URL
package shortcode

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// Ссылка имеет фиксированную длину 10 символов
const Length = 10

// Константа alphabet содержит все валидные символы для генерации:
const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"

const alphabetLen = uint64(len(alphabet))

// Generate создает детерминированный код из ссылки
// Параметр attempt используется для обработки возможных коллизий:
// attempt=0 для первой попытки, attempt=1,2,... при обнаружении коллизии.
func Generate(url string, attempt int) string {
	input := url
	if attempt > 0 {
		input = fmt.Sprintf("%s#%d", url, attempt)
	}

	hash := sha256.Sum256([]byte(input))

	// Взять первые 8 байт хэша для шифрования
	num := binary.BigEndian.Uint64(hash[:8])

	return encode(num)
}

// encode конвертирует uint64 число в base63 строку фиксированной длины.
func encode(num uint64) string {
	result := make([]byte, Length)

	for i := Length - 1; i >= 0; i-- {
		result[i] = alphabet[num%alphabetLen]
		num /= alphabetLen
	}

	return string(result)
}

// IsValid проверяет, является ли строка валидным коротким кодом
func IsValid(code string) bool {
	if len(code) != Length {
		return false
	}

	for _, c := range code {
		if !isValidChar(c) {
			return false
		}
	}

	return true
}

// isValidChar проверяет наличие символа в алфавите
func isValidChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}
