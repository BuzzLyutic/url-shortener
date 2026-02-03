.PHONY: build run test lint clean docker-build docker-up docker-down

# Сборка бинарника
build:
	go build -o bin/shortener ./cmd/shortener

# Запуск с хранилищем в памяти приложения
run:
	go run ./cmd/shortener --storage=memory

# Запуск с Postgres
run-postgres:
	go run ./cmd/shortener --storage=postgres --database-url="postgres://test:test@localhost:5432/shortener?sslmode=disable"

# Запуск всех тестов
test:
	go test ./... -v -race -cover

# Запуск тестов с отчетом о покрытии
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Запуск бенчмарков
bench:
	go test ./... -bench=. -benchmem

# Lint
lint:
	golangci-lint run

# Очистка артефактов сборки
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Запуск Postgres (для разработки)
postgres-up:
	docker run --name postgres-dev -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=shortener -p 5432:5432 -d postgres:15-alpine
	sleep 3
	docker exec -i postgres-dev psql -U test -d shortener < migrations/000001_init.up.sql

# Остановка Postgres
postgres-down:
	docker stop postgres-dev && docker rm postgres-dev

# Команды Docker
docker-build:
	docker build -t url-shortener .

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f app

# Запуск с хранилищем в памяти в Docker
docker-up-memory:
	docker-compose -f docker-compose.memory.yml up --build -d

docker-down-memory:
	docker-compose -f docker-compose.memory.yml down
