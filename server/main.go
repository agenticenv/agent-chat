package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"
	"github.com/agenticenv/agent-sdk-go/pkg/interfaces"
	"github.com/agenticenv/agent-sdk-go/pkg/llm"
	"github.com/agenticenv/agent-sdk-go/pkg/llm/anthropic"
	"github.com/agenticenv/agent-sdk-go/pkg/llm/gemini"
	"github.com/agenticenv/agent-sdk-go/pkg/llm/openai"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	agentconv "github.com/agenticenv/agent-chat/server/agent"
	"github.com/agenticenv/agent-chat/server/config"
	"github.com/agenticenv/agent-chat/server/db"
	"github.com/agenticenv/agent-chat/server/handlers"
	"github.com/agenticenv/agent-chat/server/store"
)

func main() {
	ctx := context.Background()

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	initSlog(cfg.LogLevel)
	slog.Info("starting agent-chat server", "log_level", cfg.LogLevel, "agent", cfg.Agent.Name)

	// ── Database ──────────────────────────────────────────────────────────────
	pool, err := db.Connect(ctx, cfg.DB.URL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	slog.Info("database ready")

	// ── Stores ────────────────────────────────────────────────────────────────
	convStore := store.NewConversationStore(pool)
	msgStore := store.NewMessageStore(pool)
	pgConv := agentconv.NewPGConversation(msgStore)

	// ── LLM client ───────────────────────────────────────────────────────────
	llmClient, err := getLLMClient(cfg)
	if err != nil {
		log.Fatalf("llm client: %v", err)
	}

	// ── Agent (SDK) ──────────────────────────────────────────────────────────
	// The SDK handles all Temporal internals: client, worker, workflows, activities.
	a, err := sdkagent.NewAgent(
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
		sdkagent.WithConversation(pgConv),
		sdkagent.WithConversationSize(cfg.Agent.ConvWindowSize),
		sdkagent.WithToolApprovalPolicy(sdkagent.AutoToolApprovalPolicy()),
	)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}
	defer a.Close()
	slog.Info("agent ready", "agent", cfg.Agent.Name)

	// ── Handlers ──────────────────────────────────────────────────────────────
	convH := handlers.NewConversationHandler(convStore)
	msgH := handlers.NewMessageHandler(msgStore, convStore, a)

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Route("/api", func(r chi.Router) {
		r.Get("/conversations", convH.List)
		r.Post("/conversations", convH.Create)
		r.Patch("/conversations/{id}", convH.Update)
		r.Delete("/conversations/{id}", convH.Delete)
		r.Get("/conversations/{id}/messages", msgH.List)
		r.Post("/conversations/{id}/messages", msgH.Send)
	})

	// ── HTTP server with graceful shutdown ────────────────────────────────────
	srv := &http.Server{
		Addr:    ":" + config.HTTPListenPort,
		Handler: r,
		// WriteTimeout must cover the full Temporal + LLM round trip.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Minute,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("http server listening", "addr", ":"+config.HTTPListenPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-quit
	slog.Info("shutting down")

	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

func initSlog(level string) {
	lvl := parseLogLevel(level)
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(h))
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getLLMClient(cfg *config.Config) (interfaces.LLMClient, error) {
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

// corsMiddleware allows the UI (different port in dev) to call the API.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
