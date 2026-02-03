// Пакет config предоставляет конфигурацию для приложения.
package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Config содержит конфиг приложения
type Config struct {
	// Настройки сервера
	ServerAddress string
	BaseURL       string

	// Настройки хранилища
	StorageType string // либо в памяти приложения, либо Postgres
	DatabaseURL string

	// Настройки URL
	DefaultTTL time.Duration

	// Логирование
	LogLevel string
}

// Load загружает конфиг из флагов и переменных окружения
func Load() (*Config, error) {
	cfg := &Config{}

	// Определение флагов
	flag.StringVar(&cfg.ServerAddress, "address", ":8080", "Server address (HOST:PORT)")
	flag.StringVar(&cfg.BaseURL, "base-url", "http://localhost:8080", "Base URL for short links")
	flag.StringVar(&cfg.StorageType, "storage", "memory", "Storage type: memory or postgres")
	flag.StringVar(&cfg.DatabaseURL, "database-url", "", "PostgreSQL connection string")
	flag.DurationVar(&cfg.DefaultTTL, "ttl", 0, "Default TTL for links (0 = no expiration)")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level: debug, info, warn, error")

	flag.Parse()

	// Переопределение переменными окружения
	if env := os.Getenv("SERVER_ADDRESS"); env != "" {
		cfg.ServerAddress = env
	}
	if env := os.Getenv("BASE_URL"); env != "" {
		cfg.BaseURL = env
	}
	if env := os.Getenv("STORAGE_TYPE"); env != "" {
		cfg.StorageType = env
	}
	if env := os.Getenv("DATABASE_URL"); env != "" {
		cfg.DatabaseURL = env
	}
	if env := os.Getenv("DEFAULT_TTL"); env != "" {
		ttl, err := time.ParseDuration(env)
		if err != nil {
			return nil, fmt.Errorf("invalid DEFAULT_TTL: %w", err)
		}
		cfg.DefaultTTL = ttl
	}
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		cfg.LogLevel = env
	}

	// Валидация
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	if c.StorageType != "memory" && c.StorageType != "postgres" {
		return fmt.Errorf("invalid storage type: %s (must be 'memory' or 'postgres')", c.StorageType)
	}

	if c.StorageType == "postgres" && c.DatabaseURL == "" {
		return fmt.Errorf("database-url is required when storage=postgres")
	}

	return nil
}
