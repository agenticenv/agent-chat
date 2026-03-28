# agent-demo

Agent demo — a sample app built with [agent-sdk-go](https://github.com/vvsynapse/agent-sdk-go) (Powered by [Temporal](https://temporal.io)). It includes a chat-style React UI and a Go server with REST APIs, aimed at single-agent chat today and **multi-agent** selection, routing, and orchestration demos as the project grows.

## Overview

**agent-demo** uses [agent-sdk-go](https://github.com/vvsynapse/agent-sdk-go) (Powered by [Temporal](https://temporal.io)) for durable, workflow-orchestrated conversations. Today it provides a general chat experience; the same codebase is meant to extend with agent selection and multi-agent orchestration examples.

## Built with

- **[agent-sdk-go](https://github.com/vvsynapse/agent-sdk-go)** — AI agent SDK for Go (Powered by [Temporal](https://temporal.io))
- **React Router 7 + Vite** — Single-page app UI
- **Tailwind CSS v4** — Styling

## Setup

### Run the UI locally

```bash
cd ui
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173). The UI fetches data from the backend (proxied via `/api` in dev). Start the server for full functionality.

### Environment variables

| Variable | Where | Description |
|----------|-------|-------------|
| `API_PROXY_TARGET` | Local dev (`.env`) | Backend URL for Vite proxy. Default `http://localhost:8080`. Use when backend runs on another port. |
| `API_BASE` | Docker / docker-compose | API base URL at runtime. Default `/api` (same-origin). Use full URL when backend is elsewhere. |

**Local dev — backend on different port:**

```bash
# ui/.env
API_PROXY_TARGET=http://localhost:3001
```

Restart `npm run dev` after changing.

**Docker — point to backend:**

```bash
docker run -p 3000:3000 -e API_BASE=http://localhost:8080/api agent-demo-ui
```

Or in `docker-compose.yml`:

```yaml
environment:
  API_BASE: ${API_BASE:-http://localhost:8080/api}
```

### Run the server

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

```bash
docker build -t agent-demo-ui ./ui
```

Runtime env vars: `API_BASE`, `PORT`. See [Environment variables](#environment-variables) above.
