# URL Shortener

![CI](https://github.com/BuzzLyutic/url-shortener/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
[![codecov](https://codecov.io/gh/BuzzLyutic/url-shortener/branch/main/graph/badge.svg)](https://codecov.io/gh/BuzzLyutic/url-shortener)

Сервис сокращения ссылок на Go.

## Статус разработки

**В разработке**

## Требования

- Go 1.25
- PostgreSQL 15+
- Docker & Docker Compose

## Быстрый старт

### Docker Compose (рекомендуется)
```bash
# Запуск с PostgreSQL
docker-compose up --build -d

# Проверка
curl http://localhost:8080/health

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


```

### Переход по короткой ссылке
```bash

```

## Архитектура
TODO: Добавить диаграмму

## Лицензия
MIT
