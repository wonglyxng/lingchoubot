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
	Enabled         bool
	FallbackEnabled bool // 当 LLM 调用失败时，是否自动降级到 MockRunner
	BaseURL         string
	APIKey          string
	Model           string
	Roles           map[string]LLMRoleConfig // 按角色覆盖，key 为 "pm"/"supervisor"/"worker"/"reviewer"
}

// LLMRoleConfig 是单个角色的 LLM 覆盖配置，空字段回退到全局 LLMConfig。
type LLMRoleConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

// ResolveForRole 返回指定角色的最终 LLM 配置（角色覆盖 > 全局默认）。
func (c LLMConfig) ResolveForRole(role string) (baseURL, apiKey, model string) {
	baseURL, apiKey, model = c.BaseURL, c.APIKey, c.Model
	if rc, ok := c.Roles[role]; ok {
		if rc.BaseURL != "" {
			baseURL = rc.BaseURL
		}
		if rc.APIKey != "" {
			apiKey = rc.APIKey
		}
		if rc.Model != "" {
			model = rc.Model
		}
	}
	return
}

type TemporalConfig struct {
	Enabled        bool
	HostPort       string
	Namespace      string
	TaskQueue      string
	WorkerEmbedded bool // true = embed worker in API process (dev mode), false = separate worker process
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
		LLM: loadLLMConfig(),
		Temporal: TemporalConfig{
			Enabled:        getEnv("TEMPORAL_ENABLED", "false") == "true",
			HostPort:       getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
			Namespace:      getEnv("TEMPORAL_NAMESPACE", "default"),
			TaskQueue:      getEnv("TEMPORAL_TASK_QUEUE", "lingchou-orchestrator"),
			WorkerEmbedded: getEnv("TEMPORAL_WORKER_EMBEDDED", "true") == "true",
		},
	}
}

func loadLLMConfig() LLMConfig {
	globalBaseURL := getEnv("LLM_BASE_URL", "https://api.openai.com/v1")
	globalAPIKey := getEnv("LLM_API_KEY", "")
	globalModel := getEnv("LLM_MODEL", "gpt-4o-mini")

	roleKeys := []struct {
		envPrefix string
		key       string
	}{
		{"LLM_PM_", "pm"},
		{"LLM_SUPERVISOR_", "supervisor"},
		{"LLM_WORKER_", "worker"},
		{"LLM_REVIEWER_", "reviewer"},
	}

	roles := make(map[string]LLMRoleConfig)
	for _, rk := range roleKeys {
		rc := LLMRoleConfig{
			BaseURL: getEnv(rk.envPrefix+"BASE_URL", ""),
			APIKey:  getEnv(rk.envPrefix+"API_KEY", ""),
			Model:   getEnv(rk.envPrefix+"MODEL", ""),
		}
		if rc.BaseURL != "" || rc.APIKey != "" || rc.Model != "" {
			roles[rk.key] = rc
		}
	}

	return LLMConfig{
		Enabled:         getEnv("LLM_ENABLED", "false") == "true",
		FallbackEnabled: getEnv("LLM_FALLBACK_ENABLED", "true") == "true",
		BaseURL:         globalBaseURL,
		APIKey:          globalAPIKey,
		Model:           globalModel,
		Roles:           roles,
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
