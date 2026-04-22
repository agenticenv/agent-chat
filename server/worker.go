package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	sdkagent "github.com/agenticenv/agent-sdk-go/pkg/agent"

	agentconv "github.com/agenticenv/agent-chat/server/agent"
	"github.com/agenticenv/agent-chat/server/agentsetup"
	"github.com/agenticenv/agent-chat/server/config"
	"github.com/agenticenv/agent-chat/server/db"
	"github.com/agenticenv/agent-chat/server/store"
)

// runWorker starts the Temporal worker. It polls the shared task queue and
// executes agent workflows (LLM calls, tool routing, message persistence).
// Postgres is required because the SDK's conversation activities run in this
// process and read/write messages via PGConversation.
func runWorker() {
	ctx := context.Background()

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	initSlog(cfg.LogLevel)
	slog.Info("starting agent-chat worker", "log_level", cfg.LogLevel, "agent", cfg.Agent.Name)

	// ── Database ──────────────────────────────────────────────────────────────
	// Migrations are owned by the API server; the worker only reads/writes.
	pool, err := db.Connect(ctx, cfg.DB.URL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	// ── Stores ────────────────────────────────────────────────────────────────
	msgStore := store.NewMessageStore(pool)
	pgConv := agentconv.NewPGConversation(msgStore)

	// ── Agent worker (SDK) ───────────────────────────────────────────────────
	// CommonOptions must produce an identical option set to the API server so
	// the SDK fingerprints match and workflows are accepted.
	opts, err := agentsetup.CommonOptions(cfg, pgConv)
	if err != nil {
		log.Fatalf("agent setup: %v", err)
	}

	w, err := sdkagent.NewAgentWorker(opts...)
	if err != nil {
		log.Fatalf("worker: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("worker starting", "task_queue", cfg.Temporal.TaskQueue)
	go func() {
		if err := w.Start(ctx); err != nil {
			log.Fatalf("worker stopped: %v", err)
		}
	}()

	<-quit
	slog.Info("shutting down worker")
	w.Stop()
}
