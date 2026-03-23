# ai-assistant-demo

AI Assistant Demo — an AI assistant application built on [Temporal](https://temporal.io) using the [temporal-agent-sdk-go](https://github.com/vvsynapse/temporal-agent-sdk-go) SDK.

## Overview

**ai-assistant-demo** is an AI assistant that leverages [temporal-agent-sdk-go](https://github.com/vvsynapse/temporal-agent-sdk-go) for durable, workflow-orchestrated conversations. It includes a server, a web UI, and runs Temporal via Docker Compose.

## Built with

- **[temporal-agent-sdk-go](https://github.com/vvsynapse/temporal-agent-sdk-go)** — Temporal-native AI agent SDK for Go

## Running with Docker

### Start all services (server, UI, Temporal)

```bash
# Clone the repository
git clone https://github.com/vvsynapse/ai-assistant-demo.git
cd ai-assistant-demo

# Start server, UI, and Temporal
docker compose up -d

# View logs
docker compose logs -f
```

### Start individual services

```bash
# Server only (requires Temporal to be running)
docker compose up -d temporal
docker compose up -d server

# UI only
docker compose up -d ui
```

### Stop services

```bash
docker compose down
```

## Project structure

```
ai-assistant-demo/
├── docker-compose.yml   # Server, UI, and Temporal
├── server/             # Go module (own go.mod)
│   ├── go.mod
│   ├── main.go
│   └── Dockerfile
├── ui/
│   ├── Dockerfile
│   └── public/         # Static UI assets
│       └── index.html
└── README.md
```
