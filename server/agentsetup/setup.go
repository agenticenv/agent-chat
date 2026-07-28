// Package agentsetup builds the shared set of agent SDK options used by both
// the HTTP API server and the Temporal worker entrypoints.
//
// Both processes must produce identical agent fingerprints (same name,
// description, system prompt, LLM client, timeout, etc.) so the SDK accepts
// the worker as a valid executor for workflows submitted by the API server.
// Centralizing option construction here prevents drift between the two.
package agentsetup

import (
	"errors"

	"github.com/agenticenv/agent-chat/server/config"
	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"
	"github.com/agenticenv/agent-sdk-go/pkg/conversation"
	"github.com/agenticenv/agent-sdk-go/pkg/interfaces"
	"github.com/agenticenv/agent-sdk-go/pkg/llm"
	"github.com/agenticenv/agent-sdk-go/pkg/llm/anthropic"
	"github.com/agenticenv/agent-sdk-go/pkg/llm/gemini"
	"github.com/agenticenv/agent-sdk-go/pkg/llm/openai"
)

// CommonOptions builds the agent option slice shared by the API server and
// the worker. Callers append entrypoint-specific options (e.g. the API server
// adds DisableLocalWorker + EnableRemoteWorkers).
func CommonOptions(cfg *config.Config, conv interfaces.Conversation) ([]sdkagent.Option, error) {
	llmClient, err := NewLLMClient(cfg)
	if err != nil {
		return nil, err
	}

	return []sdkagent.Option{
		sdkagent.WithTemporalConfig(&sdkagent.TemporalConfig{
			Host:      cfg.Temporal.Host,
			Port:      cfg.Temporal.Port,
			Namespace: cfg.Temporal.Namespace,
			TaskQueue: cfg.Temporal.TaskQueue,
		}),
		sdkagent.WithName(cfg.Agent.Name),
		sdkagent.WithDescription(cfg.Agent.Description),
		sdkagent.WithSystemPrompt(cfg.Agent.SystemPrompt),
		sdkagent.WithLogLevel(cfg.LogLevel),
		sdkagent.WithLLMClient(llmClient),
		sdkagent.WithConversation(conversation.Config{
			Conversation: conv,
			Size:         cfg.Agent.ConvWindowSize,
		}),
		sdkagent.WithToolApprovalPolicy(sdkagent.AutoToolApprovalPolicy()),
	}, nil
}

// NewLLMClient constructs the configured LLM client. Exported so entrypoints
// can build one directly when they need the client outside of CommonOptions.
func NewLLMClient(cfg *config.Config) (interfaces.LLMClient, error) {
	switch cfg.LLM.Provider {
	case "openai":
		return openai.NewClient(
			llm.WithAPIKey(cfg.LLM.APIKey),
			llm.WithModel(cfg.LLM.Model),
			llm.WithBaseURL(cfg.LLM.BaseURL),
		)
	case "anthropic":
		return anthropic.NewClient(
			llm.WithAPIKey(cfg.LLM.APIKey),
			llm.WithModel(cfg.LLM.Model),
		)
	case "gemini":
		return gemini.NewClient(
			llm.WithAPIKey(cfg.LLM.APIKey),
			llm.WithModel(cfg.LLM.Model),
		)
	default:
		return nil, errors.New("invalid LLM provider")
	}
}
