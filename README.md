# URL Shortener

![CI](https://github.com/BuzzLyutic/url-shortener/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)
[![codecov](https://codecov.io/gh/BuzzLyutic/url-shortener/branch/main/graph/badge.svg)](https://codecov.io/gh/BuzzLyutic/url-shortener)

Сервис сокращения ссылок на Go.

---

## Возможности

- Сокращение длинных URL до 10-символьного кода
- Детерминированная генерация (один URL = один код)
- Два хранилища: PostgreSQL и in-memory
- REST API
- Docker-образ для деплоя
- Graceful shutdown
- TTL для ссылок (опционально)

---


## Быстрый старт

### Docker Compose (рекомендуется)
```bash
# Запуск с PostgreSQL
docker-compose up --build -d

# Проверка
curl http://localhost:8080/health

# Создание короткой ссылки
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/path"}'

# Остановка
docker-compose down
```

### Docker (только memory storage)
```bash
docker-compose -f docker-compose.memory.yml up --build -d
```

### Локальный запуск
```bash

# С in-memory хранилищем
go run ./cmd/shortener --storage=memory

# С PostgreSQL (требуется запущенный PostgreSQL)
go run ./cmd/shortener \
  --storage=postgres \
  --database-url="postgres://user:pass@localhost:5432/shortener?sslmode=disable"
```

## Конфигурация
| Переменная |	Флаг |	Описание |	По умолчанию |
| - | - | - | - |
| SERVER_ADDRESS |	--address |	Адрес сервера |	:8080 |
BASE_URL |	--base-url |	Базовый URL для коротких ссылок |	http://localhost:8080
STORAGE_TYPE |	--storage |	Тип хранилища: memory или postgres |	memory
DATABASE_URL |	--database-url |	Строка подключения PostgreSQL |	-
DEFAULT_TTL |	--ttl |	TTL для ссылок (например: 24h) |	0 (бессрочно)
LOG_LEVEL |	--log-level |	Уровень логов: debug, info, warn, error	| info |

## API

### Создание короткой ссылки

```bash
POST /api/shorten
Content-Type: application/json

{"url": "https://example.com/very/long/path"}
```
### Ответ (201 Created / 200 OK):

```bash
{
  "short_url": "http://localhost:8080/aB3_xY9z12",
  "original_url": "https://example.com/very/long/path",
  "expires_at": null
}
```
### Переход по короткой ссылке
```bash
GET /{code}
```

### Ошибки
| Код |	Ошибка |	Описание |
| - | - | - |
400 |	empty_url |	URL пустой
400 |	invalid_url |	Невалидный формат URL
400 |	invalid_json |	Невалидный JSON
404 |	not_found |	Короткая ссылка не найдена

---

## Алгоритм генерации

![alt text](docs/image-1.png)


Используется детерминированный алгоритм на основе SHA-256:

1. Вычисляется SHA-256 хэш от URL
2. Первые 8 байт преобразуются в Base63 (a-z, A-Z, 0-9, _)
3. Берутся первые 10 символов

Преимущества:

- Один URL всегда получает один и тот же код
- Идемпотентность POST запросов
- Не требует координации между инстансами

Обработка коллизий: при совпадении хэша добавляется соль и генерируется новый код (до 10 попыток).

---

## Нагрузочное тестирование

```bash

# Установка hey
go install github.com/rakyll/hey@latest

# Тест
hey -n 10000 -c 100 http://localhost:8080/CODE
```

## Покрытие тестами

| Пакет | Покрытие |
|-|-|
| `internal/shortcode` | 100.0% |
| `internal/storage` | 29.8% |
| `internal/service` | 90.7% |
| `internal/handler` | 88.1% |
| `internal/config` | 16.1% |
| **Общее** | **46.5%** |

```bash
go test ./... -cover
```

## Разработка

```bash

# Запуск тестов
make test

# Запуск с race detector
go test ./... -race

# Линтинг
make lint

# Покрытие
make test-coverage

```

---
