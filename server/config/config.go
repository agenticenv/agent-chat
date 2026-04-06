package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// HTTPListenPort is the fixed API listen port inside the container / process.
const HTTPListenPort = "8080"

type Config struct {
	LogLevel string // slog level: debug, info, warn, error (default info)
	DB       *DBConfig
	Temporal *TemporalConfig
	LLM      *LLMConfig
	Agent    *AgentConfig
}

type DBConfig struct {
	URL string
}

type TemporalConfig struct {
	Host      string
	Port      int
	Namespace string
	TaskQueue string
}

type LLMConfig struct {
	Provider string // openai, anthropic, gemini
	APIKey   string
	Model    string
	BaseURL  string // Optional; only used when provider is openai (custom/Azure-compatible API).
}

type AgentConfig struct {
	Name           string
	Description    string
	SystemPrompt   string
	ConvWindowSize int
}

// loadDotEnv merges .env files into the process environment. Does not override existing vars
// (Docker Compose env_file is applied before the process starts, so those values win).
// If ENV_FILE is set, that file is loaded first; server/.env and .env still run after to fill gaps.
func loadDotEnv() {
	if p := os.Getenv("ENV_FILE"); p != "" {
		_ = godotenv.Load(p)
	}
	_ = godotenv.Load(filepath.Join("server", ".env"))
	_ = godotenv.Load(".env")
}

func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{
		LogLevel: strings.TrimSpace(getenvOr("LOG_LEVEL", "info")),
		DB:       &DBConfig{URL: databaseURL()},
		Temporal: &TemporalConfig{
			Host:      getenvOr("TEMPORAL_HOST", "temporal"),
			Port:      getenvIntOr("TEMPORAL_PORT", 7233),
			Namespace: getenvOr("TEMPORAL_NAMESPACE", "default"),
			TaskQueue: getenvOr("TEMPORAL_TASK_QUEUE", "agent-chat"),
		},
		LLM: &LLMConfig{
			Provider: getenvOr("LLM_PROVIDER", "openai"),
			APIKey:   os.Getenv("LLM_API_KEY"),
			Model:    getenvOr("LLM_MODEL", "gpt-4o"),
			BaseURL:  os.Getenv("LLM_BASE_URL"),
		},
		Agent: &AgentConfig{
			Name:           getenvOr("AGENT_NAME", "agent-chat"),
			Description:    getenvOr("AGENT_DESCRIPTION", "Assitant Chat Agent"),
			SystemPrompt:   getenvOr("AGENT_SYSTEM_PROMPT", "You are a helpful assistant."),
			ConvWindowSize: getenvIntOr("AGENT_CONVERSATION_WINDOW_SIZE", 20),
		},
	}

	if cfg.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required (set in server/.env or environment)")
	}

	return cfg, nil
}

// databaseURL returns DATABASE_URL if set, otherwise builds postgres:// from POSTGRES_* (Compose / shell).
func databaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	user := getenvOr("POSTGRES_USER", "temporal")
	pass := getenvOr("POSTGRES_PASSWORD", "temporal")
	host := getenvOr("POSTGRES_HOST", "localhost")
	port := getenvOr("POSTGRES_PORT", "5432")
	dbname := getenvOr("POSTGRES_DB", "agentdb")

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, pass),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + dbname,
	}
	q := url.Values{}
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

func getenvOr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getenvIntOr(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
