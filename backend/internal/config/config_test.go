package config

import (
	"os"
	"testing"
)

func TestResolveForRole_GlobalDefaults(t *testing.T) {
	cfg := LLMConfig{
		BaseURL: "http://global.example.com",
		APIKey:  "global-key",
		Model:   "gpt-4o",
		Roles:   map[string]LLMRoleConfig{},
	}

	baseURL, apiKey, model := cfg.ResolveForRole("pm")
	if baseURL != "http://global.example.com" {
		t.Errorf("expected global base URL, got %s", baseURL)
	}
	if apiKey != "global-key" {
		t.Errorf("expected global API key, got %s", apiKey)
	}
	if model != "gpt-4o" {
		t.Errorf("expected global model, got %s", model)
	}
}

func TestResolveForRole_RoleOverride(t *testing.T) {
	cfg := LLMConfig{
		BaseURL: "http://global.example.com",
		APIKey:  "global-key",
		Model:   "gpt-4o",
		Roles: map[string]LLMRoleConfig{
			"pm": {
				BaseURL: "http://pm.example.com",
				Model:   "gpt-4o-mini",
			},
		},
	}

	baseURL, apiKey, model := cfg.ResolveForRole("pm")
	if baseURL != "http://pm.example.com" {
		t.Errorf("expected PM base URL, got %s", baseURL)
	}
	if apiKey != "global-key" {
		t.Errorf("expected global API key (no PM override), got %s", apiKey)
	}
	if model != "gpt-4o-mini" {
		t.Errorf("expected PM model, got %s", model)
	}
}

func TestResolveForRole_FullOverride(t *testing.T) {
	cfg := LLMConfig{
		BaseURL: "http://global.example.com",
		APIKey:  "global-key",
		Model:   "gpt-4o",
		Roles: map[string]LLMRoleConfig{
			"reviewer": {
				BaseURL: "http://reviewer.example.com",
				APIKey:  "reviewer-key",
				Model:   "claude-3",
			},
		},
	}

	baseURL, apiKey, model := cfg.ResolveForRole("reviewer")
	if baseURL != "http://reviewer.example.com" {
		t.Errorf("expected reviewer base URL, got %s", baseURL)
	}
	if apiKey != "reviewer-key" {
		t.Errorf("expected reviewer API key, got %s", apiKey)
	}
	if model != "claude-3" {
		t.Errorf("expected reviewer model, got %s", model)
	}
}

func TestResolveForRole_UnknownRole(t *testing.T) {
	cfg := LLMConfig{
		BaseURL: "http://global.example.com",
		APIKey:  "global-key",
		Model:   "gpt-4o",
		Roles: map[string]LLMRoleConfig{
			"pm": {Model: "pm-model"},
		},
	}

	baseURL, apiKey, model := cfg.ResolveForRole("unknown")
	if baseURL != "http://global.example.com" {
		t.Errorf("expected global base URL for unknown role, got %s", baseURL)
	}
	if apiKey != "global-key" {
		t.Errorf("expected global API key for unknown role, got %s", apiKey)
	}
	if model != "gpt-4o" {
		t.Errorf("expected global model for unknown role, got %s", model)
	}
}

func TestResolveForRole_NilRoles(t *testing.T) {
	cfg := LLMConfig{
		BaseURL: "http://global.example.com",
		APIKey:  "global-key",
		Model:   "gpt-4o",
		Roles:   nil,
	}

	baseURL, apiKey, model := cfg.ResolveForRole("pm")
	if baseURL != "http://global.example.com" {
		t.Errorf("expected global base URL, got %s", baseURL)
	}
	if apiKey != "global-key" {
		t.Errorf("expected global API key, got %s", apiKey)
	}
	if model != "gpt-4o" {
		t.Errorf("expected global model, got %s", model)
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear environment to test defaults
	envKeys := []string{
		"SERVER_HOST", "SERVER_PORT", "DB_HOST", "DB_PORT", "DB_USER",
		"DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "API_KEY",
		"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY",
		"MINIO_BUCKET", "MINIO_USE_SSL",
		"LLM_ENABLED", "LLM_BASE_URL", "LLM_API_KEY", "LLM_MODEL",
		"TEMPORAL_ENABLED", "TEMPORAL_HOST_PORT", "TEMPORAL_NAMESPACE", "TEMPORAL_TASK_QUEUE",
	}
	saved := make(map[string]string)
	for _, k := range envKeys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	cfg := Load()

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("expected default DB host localhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("expected default DB port 5432, got %d", cfg.Database.Port)
	}
	if cfg.LLM.Enabled {
		t.Error("LLM should be disabled by default")
	}
	if !cfg.LLM.FallbackEnabled {
		t.Error("LLM fallback should be enabled by default")
	}
	if cfg.LLM.Model != "gpt-4o-mini" {
		t.Errorf("expected default LLM model gpt-4o-mini, got %s", cfg.LLM.Model)
	}
	if cfg.Temporal.Enabled {
		t.Error("Temporal should be disabled by default")
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	envs := map[string]string{
		"SERVER_PORT":    "9090",
		"LLM_ENABLED":   "true",
		"LLM_BASE_URL":  "http://custom-llm.com/v1",
		"LLM_API_KEY":   "sk-test",
		"LLM_MODEL":     "gpt-4o",
		"LLM_PM_MODEL":  "claude-3",
		"API_KEY":        "my-api-key",
	}

	// Save and set
	saved := make(map[string]string)
	for k, v := range envs {
		saved[k] = os.Getenv(k)
		os.Setenv(k, v)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	cfg := Load()

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if !cfg.LLM.Enabled {
		t.Error("LLM should be enabled")
	}
	if cfg.LLM.BaseURL != "http://custom-llm.com/v1" {
		t.Errorf("expected custom LLM URL, got %s", cfg.LLM.BaseURL)
	}
	if cfg.LLM.APIKey != "sk-test" {
		t.Errorf("expected LLM API key sk-test, got %s", cfg.LLM.APIKey)
	}
	if cfg.LLM.Model != "gpt-4o" {
		t.Errorf("expected LLM model gpt-4o, got %s", cfg.LLM.Model)
	}
	if cfg.APIKey != "my-api-key" {
		t.Errorf("expected API key, got %s", cfg.APIKey)
	}

	// Check per-role config
	pmRole, ok := cfg.LLM.Roles["pm"]
	if !ok {
		t.Fatal("expected PM role config")
	}
	if pmRole.Model != "claude-3" {
		t.Errorf("expected PM model claude-3, got %s", pmRole.Model)
	}

	// Verify resolve
	_, _, model := cfg.LLM.ResolveForRole("pm")
	if model != "claude-3" {
		t.Errorf("expected resolved PM model claude-3, got %s", model)
	}
	_, _, model = cfg.LLM.ResolveForRole("supervisor")
	if model != "gpt-4o" {
		t.Errorf("expected resolved supervisor model gpt-4o (global), got %s", model)
	}
}

func TestDatabaseDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "db.example.com",
		Port:     5433,
		User:     "admin",
		Password: "secret",
		DBName:   "mydb",
		SSLMode:  "require",
	}

	dsn := cfg.DSN()
	expected := "postgres://admin:secret@db.example.com:5433/mydb?sslmode=require"
	if dsn != expected {
		t.Errorf("expected DSN %q, got %q", expected, dsn)
	}
}

func TestLoadLLMConfig_RoleOverrides(t *testing.T) {
	envs := map[string]string{
		"LLM_BASE_URL":           "http://global.com/v1",
		"LLM_API_KEY":            "global-key",
		"LLM_MODEL":              "gpt-4o",
		"LLM_PM_BASE_URL":        "http://pm.com/v1",
		"LLM_PM_API_KEY":         "pm-key",
		"LLM_PM_MODEL":           "pm-model",
		"LLM_SUPERVISOR_MODEL":   "sup-model",
		"LLM_WORKER_BASE_URL":    "http://worker.com/v1",
	}

	saved := make(map[string]string)
	for k, v := range envs {
		saved[k] = os.Getenv(k)
		os.Setenv(k, v)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	cfg := loadLLMConfig()

	// PM: all three overridden
	pmRole := cfg.Roles["pm"]
	if pmRole.BaseURL != "http://pm.com/v1" || pmRole.APIKey != "pm-key" || pmRole.Model != "pm-model" {
		t.Errorf("PM role config unexpected: %+v", pmRole)
	}

	// Supervisor: only model overridden
	supRole := cfg.Roles["supervisor"]
	if supRole.Model != "sup-model" {
		t.Errorf("supervisor model expected sup-model, got %s", supRole.Model)
	}
	if supRole.BaseURL != "" || supRole.APIKey != "" {
		t.Errorf("supervisor should not have base_url/api_key override: %+v", supRole)
	}

	// Worker: only base_url overridden
	workerRole := cfg.Roles["worker"]
	if workerRole.BaseURL != "http://worker.com/v1" {
		t.Errorf("worker base_url expected override, got %s", workerRole.BaseURL)
	}

	// Reviewer: no overrides
	if _, ok := cfg.Roles["reviewer"]; ok {
		t.Error("reviewer should not be in roles map (no overrides)")
	}

	// Resolve PM → all overridden
	baseURL, apiKey, model := cfg.ResolveForRole("pm")
	if baseURL != "http://pm.com/v1" || apiKey != "pm-key" || model != "pm-model" {
		t.Errorf("PM resolve unexpected: %s, %s, %s", baseURL, apiKey, model)
	}

	// Resolve supervisor → model overridden, rest global
	baseURL, apiKey, model = cfg.ResolveForRole("supervisor")
	if baseURL != "http://global.com/v1" || apiKey != "global-key" || model != "sup-model" {
		t.Errorf("supervisor resolve unexpected: %s, %s, %s", baseURL, apiKey, model)
	}
}
