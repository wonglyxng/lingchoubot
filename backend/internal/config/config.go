package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	MinIO    MinIOConfig
	LLM      LLMConfig
	Temporal TemporalConfig
	APIKey   string
}

type LLMConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Model   string
}

type TemporalConfig struct {
	Enabled   bool
	HostPort  string
	Namespace string
	TaskQueue string
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

func Load() *Config {
	return &Config{
		APIKey: getEnv("API_KEY", ""),
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "lingchou"),
			Password: getEnv("DB_PASSWORD", "lingchou"),
			DBName:   getEnv("DB_NAME", "lingchou"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "lingchou"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "lingchou_minio_dev"),
			Bucket:    getEnv("MINIO_BUCKET", "lingchou-artifacts"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		},
		LLM: LLMConfig{
			Enabled: getEnv("LLM_ENABLED", "false") == "true",
			BaseURL: getEnv("LLM_BASE_URL", "https://api.openai.com/v1"),
			APIKey:  getEnv("LLM_API_KEY", ""),
			Model:   getEnv("LLM_MODEL", "gpt-4o-mini"),
		},
		Temporal: TemporalConfig{
			Enabled:   getEnv("TEMPORAL_ENABLED", "false") == "true",
			HostPort:  getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
			TaskQueue: getEnv("TEMPORAL_TASK_QUEUE", "lingchou-orchestrator"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
