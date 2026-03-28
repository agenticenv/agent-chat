# agent-demo

Agent demo — a sample app built with [agent-sdk-go](https://github.com/vvsynapse/agent-sdk-go) (Powered by [Temporal](https://temporal.io)). It includes a chat-style React UI and a Go server with REST APIs, aimed at single-agent chat today and **multi-agent** selection, routing, and orchestration demos as the project grows.

## Overview

**agent-demo** uses [agent-sdk-go](https://github.com/vvsynapse/agent-sdk-go) (Powered by [Temporal](https://temporal.io)) for durable, workflow-orchestrated conversations. Today it provides a general chat experience; the same codebase is meant to extend with agent selection and multi-agent orchestration examples.

## Built with

- **[agent-sdk-go](https://github.com/vvsynapse/agent-sdk-go)** — AI agent SDK for Go (Powered by [Temporal](https://temporal.io))
- **React Router 7 + Vite** — Single-page app UI
- **Tailwind CSS v4** — Styling

## Setup

### UI (local dev & Docker)

See **[ui/README.md](ui/README.md)** for `npm run dev`, Vite proxy / `API_PROXY_TARGET`, and building or running the UI Docker image (`API_BASE`, `PORT`).

### Server

See **[server/README.md](server/README.md)** for environment variables, API endpoints, architecture details, and running the server with Docker.

```bash
docker compose up -d temporal postgres
docker compose up -d server
```

### API contract

The UI expects these REST endpoints:

| Endpoint | Method | Request | Response |
|----------|--------|---------|----------|
| `/api/conversations` | GET | - | `Conversation[]` or `{ conversations: [...] }` |
| `/api/conversations` | POST | `{ title?: string }` | `{ id, title }` or `{ conversation: {...} }` |
| `/api/conversations/:id/messages` | GET | - | `Message[]` or `{ messages: [...] }` |
| `/api/conversations/:id/messages` | POST | `{ content: string }` | `Message` or `{ message: {...} }` |

## Running with Docker

```bash
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

### Building the UI image

See **[ui/README.md](ui/README.md#docker)** (build and runtime env vars).
