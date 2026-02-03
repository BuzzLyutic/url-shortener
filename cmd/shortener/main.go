package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BuzzLyutic/url-shortener/internal/config"
	"github.com/BuzzLyutic/url-shortener/internal/handler"
	"github.com/BuzzLyutic/url-shortener/internal/service"
	"github.com/BuzzLyutic/url-shortener/internal/storage"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Установка логгера
	logger := setupLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	logger.Info("Starting URL shortener",
		slog.String("address", cfg.ServerAddress),
		slog.String("storage", cfg.StorageType),
		slog.String("base_url", cfg.BaseURL),
	)

	// Инициализация хранилища
	store, err := initStorage(cfg, logger)
	if err != nil {
		return err
	}
	defer store.Close()

	// Инициализация сервиса
	svc := service.New(store, service.Config{
		BaseURL:    cfg.BaseURL,
		DefaultTTL: cfg.DefaultTTL,
	})

	// Инициализация хэндлера
	h := handler.New(svc, logger)

	// Установка HTTP сервера
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Применить middleware
	var httpHandler http.Handler = mux
	httpHandler = handler.Logging(logger)(httpHandler)
	httpHandler = handler.Recovery(logger)(httpHandler)

	server := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	return runServer(server, logger)
}

func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: logLevel}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func initStorage(cfg *config.Config, logger *slog.Logger) (storage.Storage, error) {
	switch cfg.StorageType {
	case "postgres":
		logger.Info("connecting to PostgreSQL", slog.String("url", maskDSN(cfg.DatabaseURL)))
		pgCfg := storage.DefaultPostgresConfig(cfg.DatabaseURL)
		return storage.NewPostgresStorage(pgCfg)
	case "memory":
		logger.Info("using in-memory storage")
		return storage.NewMemoryStorage(), nil
	default:
		return nil, errors.New("unknown storage type")
	}
}

func runServer(server *http.Server, logger *slog.Logger) error {
	// Создаем канал для получения сигналов
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Канал для получения ошибок сервера
	serverErr := make(chan error, 1)

	// Старт сервера
	go func() {
		logger.Info("server listening", slog.String("address", server.Addr))
		serverErr <- server.ListenAndServe()
	}()

	// Ожидание завершения работы или ошибки
	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case sig := <-shutdown:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))

		// Ожидание дополнительно 10 секунд для завершения обрабатываемых запросов
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			// Насильное завершение работы
			server.Close()
			return err
		}
	}
	logger.Info("server stopped")
	return nil
}

func maskDSN(dsn string) string {
	if dsn == "" {
		return "(empty)"
	}
	return "(set)"
}
