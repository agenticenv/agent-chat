package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	sdkagent "github.com/vvsynapse/temporal-agent-sdk-go/pkg/agent"
	"github.com/vvsynapse/temporal-agent-sdk-go/pkg/llm"
	"github.com/vvsynapse/temporal-agent-sdk-go/pkg/llm/openai"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	agentconv "github.com/vvsynapse/agent-demo/server/agent"
	"github.com/vvsynapse/agent-demo/server/config"
	"github.com/vvsynapse/agent-demo/server/db"
	"github.com/vvsynapse/agent-demo/server/handlers"
	"github.com/vvsynapse/agent-demo/server/store"
)

func main() {
	ctx := context.Background()

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// ── Database ──────────────────────────────────────────────────────────────
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	log.Println("database ready")

	// ── Stores ────────────────────────────────────────────────────────────────
	convStore := store.NewConversationStore(pool)
	msgStore := store.NewMessageStore(pool)
	pgConv := agentconv.NewPGConversation(msgStore)

	// ── LLM client ───────────────────────────────────────────────────────────
	llmClient, err := openai.NewClient(
		llm.WithAPIKey(cfg.AnthropicKey),
		llm.WithModel(cfg.LLMModel),
		llm.WithBaseURL(cfg.LLMBaseURL),
	)
	if err != nil {
		log.Fatalf("llm client: %v", err)
	}

	// ── Shared Temporal client ───────────────────────────────────────────────
	// One client shared between the SDK agent (activities/worker) and our
	// session workflow. We own the lifecycle; SDK does not close it.
	tc, err := client.Dial(client.Options{
		HostPort:  fmt.Sprintf("%s:%d", cfg.TemporalHost, cfg.TemporalPort),
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		log.Fatalf("temporal client: %v", err)
	}
	defer tc.Close()

	// ── Agent (SDK) ──────────────────────────────────────────────────────────
	// Creates the SDK worker that registers AgentWorkflow + all activities
	// (AgentLLMActivity, AddConversationMessagesActivity, etc.) on the task queue.
	a, err := sdkagent.NewAgent(
		sdkagent.WithTemporalClient(tc),
		sdkagent.WithTaskQueue(cfg.TaskQueue),
		sdkagent.WithSystemPrompt(cfg.SystemPrompt),
		sdkagent.WithLLMClient(llmClient),
		sdkagent.WithConversation(pgConv),
		sdkagent.WithConversationSize(cfg.ConvWindowSize),
		sdkagent.WithToolApprovalPolicy(sdkagent.AutoToolApprovalPolicy()),
	)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}
	defer a.Close()
	log.Println("agent ready")

	// ── Session worker ───────────────────────────────────────────────────────
	// Runs on a dedicated task queue so it doesn't pick up SDK activities.
	// The child AgentWorkflow is dispatched to the SDK's task queue.
	sessionTaskQueue := cfg.TaskQueue + "-sessions"
	sessionWorker := worker.New(tc, sessionTaskQueue, worker.Options{})
	sessionWorker.RegisterWorkflowWithOptions(
		agentconv.SessionWorkflow,
		workflow.RegisterOptions{Name: "SessionWorkflow"},
	)
	go func() {
		if err := sessionWorker.Start(); err != nil {
			log.Fatalf("session worker: %v", err)
		}
	}()
	defer sessionWorker.Stop()
	log.Println("session worker ready")

	// ── Handlers ──────────────────────────────────────────────────────────────
	convH := handlers.NewConversationHandler(convStore, tc)
	msgH := handlers.NewMessageHandler(msgStore, convStore, tc, sessionTaskQueue, cfg.TaskQueue)

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
		Addr:    ":" + cfg.Port,
		Handler: r,
		// WriteTimeout must cover the full Temporal + LLM round trip.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Minute,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown error: %v", err)
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
