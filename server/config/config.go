package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	AgentSDKHost      string
	AgentSDKPort      int
	AgentSDKNamespace string
	TaskQueue         string
	LLMAPIKey         string
	LLMModel          string
	LLMBaseURL        string
	SystemPrompt      string
	ConvWindowSize    int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://temporal:temporal@postgres:5432/assistant?sslmode=disable"),
		AgentSDKHost:      getEnv("AGENT_SDK_HOST", "localhost"),
		AgentSDKNamespace: getEnv("AGENT_SDK_NAMESPACE", "default"),
		TaskQueue:         getEnv("AGENT_SDK_TASK_QUEUE", "ai-assistant"),
		LLMAPIKey:         getEnv("LLM_API_KEY", ""),
		LLMModel:          getEnv("LLM_MODEL", "gpt-4o"),
		LLMBaseURL:        getEnv("LLM_BASE_URL", ""),
		SystemPrompt:      getEnv("SYSTEM_PROMPT", "You are a helpful assistant."),
	}

	var err error
	cfg.AgentSDKPort, err = strconv.Atoi(getEnv("AGENT_SDK_PORT", "7233"))
	if err != nil {
		return nil, fmt.Errorf("invalid AGENT_SDK_PORT: %w", err)
	}

	cfg.ConvWindowSize, err = strconv.Atoi(getEnv("CONVERSATION_WINDOW_SIZE", "20"))
	if err != nil {
		return nil, fmt.Errorf("invalid CONVERSATION_WINDOW_SIZE: %w", err)
	}

	if cfg.LLMAPIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}