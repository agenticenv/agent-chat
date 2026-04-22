package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	agentconv "github.com/agenticenv/agent-chat/server/agent"
	"github.com/agenticenv/agent-chat/server/agentsetup"
	"github.com/agenticenv/agent-chat/server/config"
	"github.com/agenticenv/agent-chat/server/db"
	"github.com/agenticenv/agent-chat/server/handlers"
	"github.com/agenticenv/agent-chat/server/store"
	"github.com/agenticenv/agent-chat/server/stream"
)

// runAgent starts the HTTP API server. It creates a Temporal-client-only SDK
// agent (DisableLocalWorker + EnableRemoteWorkers) — the worker process
// executes workflows on the shared task queue.
func runAgent() {
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

	// ── Agent (SDK) ──────────────────────────────────────────────────────────
	opts, err := agentsetup.CommonOptions(cfg, pgConv)
	if err != nil {
		log.Fatalf("agent setup: %v", err)
	}

	a, err := sdkagent.NewAgent(append(opts,
		sdkagent.DisableLocalWorker(),
		sdkagent.EnableRemoteWorkers(),
	)...)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}
	defer a.Close()
	slog.Info("agent ready", "agent", cfg.Agent.Name)

	// ── Stream broker + runner ────────────────────────────────────────────────
	broker := stream.NewBroker()
	runner := stream.NewRunner(a, broker, msgStore, cfg.Agent.Name, ctx)

	// ── Handlers ──────────────────────────────────────────────────────────────
	convH := handlers.NewConversationHandler(convStore)
	msgH := handlers.NewMessageHandler(msgStore, convStore, a, runner, broker)

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
		r.Post("/conversations/{id}/messages/stream", msgH.Stream)
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

	// Cancel all in-flight bridge goroutines before stopping the HTTP server.
	broker.CloseAll()

	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
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
