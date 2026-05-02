package main

import (
	"flag"
	"log"
	"log/slog"
	"os"
	"strings"
)

// main dispatches to the agent (HTTP API) or worker (Temporal worker) mode.
//
// Mode resolution order (later layers override earlier):
//  1. APP_MODE environment variable (set by docker-compose per service)
//  2. --mode CLI flag
//
// The Dockerfile bakes a default APP_MODE via ARG MODE, so a naked container
// run without env or flag still works.
func main() {
	mode := os.Getenv("APP_MODE")
	flag.StringVar(&mode, "mode", mode, "run mode: 'server' or 'worker'")
	flag.Parse()

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "server"
	}

	switch mode {
	case "server":
		runAgent()
	case "worker":
		runWorker()
	default:
		log.Fatalf("invalid mode %q (expected 'server' or 'worker')", mode)
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
