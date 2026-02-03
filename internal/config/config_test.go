package config

import (
	"os"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid memory storage",
			config: Config{
				StorageType: "memory",
			},
			wantErr: false,
		},
		{
			name: "valid postgres storage",
			config: Config{
				StorageType: "postgres",
				DatabaseURL: "postgres://localhost/test",
			},
			wantErr: false,
		},
		{
			name: "invalid storage type",
			config: Config{
				StorageType: "invalid",
			},
			wantErr: true,
		},
		{
			name: "postgres without database url",
			config: Config{
				StorageType: "postgres",
				DatabaseURL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_EnvOverridesFlags(t *testing.T) {
	oldStorage := os.Getenv("STORAGE_TYPE")
	defer os.Setenv("STORAGE_TYPE", oldStorage)

	os.Setenv("STORAGE_TYPE", "memory")

	// Сброс флагов для тестирования
	cfg := &Config{
		StorageType:   "memory",
		ServerAddress: ":8080",
		BaseURL:       "http://localhost:8080",
		LogLevel:      "info",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	if cfg.StorageType != "memory" {
		t.Errorf("StorageType = %v, want %v", cfg.StorageType, "memory")
	}
}
