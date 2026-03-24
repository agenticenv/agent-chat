# ai-assistant-demo

AI Assistant Demo — an AI assistant application built on [Temporal](https://temporal.io) using the [temporal-agent-sdk-go](https://github.com/vvsynapse/temporal-agent-sdk-go) SDK.

## Overview

**ai-assistant-demo** is an AI assistant that leverages [temporal-agent-sdk-go](https://github.com/vvsynapse/temporal-agent-sdk-go) for durable, workflow-orchestrated conversations. It includes a backend server and a React SPA that talks to it via REST APIs.

## Built with

- **[temporal-agent-sdk-go](https://github.com/vvsynapse/temporal-agent-sdk-go)** — Temporal-native AI agent SDK for Go
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
docker run -p 3000:3000 -e API_BASE=http://localhost:8080/api ai-assistant-ui
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
docker build -f ui/Dockerfile -t ai-assistant-ui .
```

Runtime env vars: `API_BASE`, `PORT`. See [Environment variables](#environment-variables) above.
