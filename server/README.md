# agent-demo Server

Go API server powered by [agent-sdk-go](https://github.com/vvsynapse/agent-sdk-go) and [Temporal](https://temporal.io). Provides REST endpoints for managing conversations and messages, with durable workflow-orchestrated AI agent execution.

## Architecture

The server embeds a Temporal worker in the same process, so no separate worker deployment is needed. Conversations and messages are persisted in PostgreSQL, and the agent SDK handles LLM orchestration through Temporal workflows.

```
server/
├── agent/        # Conversation bridge for the agent SDK
├── config/       # Environment-based configuration
├── db/           # PostgreSQL connection and migrations
├── handlers/     # HTTP route handlers
├── llm/          # Azure OpenAI client (adds api-key header + api-version param)
├── store/        # Data access layer
├── Dockerfile
└── main.go       # Entrypoint — starts HTTP server and Temporal worker
```

## Prerequisites

- Docker and Docker Compose
- An LLM API key

## Run with Docker

From the **repository root**:

```bash
# Backend only (server + its dependencies, no React UI)
# Use this when running the UI locally with npm run dev
docker compose up -d postgres temporal server
```

The server will be available at [http://localhost:8080](http://localhost:8080). Temporal UI is at [http://localhost:8233](http://localhost:8233).

```bash
# Full stack (backend + React UI + Temporal dashboard)
docker compose up -d
```

### View logs

```bash
docker compose logs -f server
```

### Stop services

```bash
docker compose down
```

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `postgres://temporal:temporal@postgres:5432/assistant?sslmode=disable` | PostgreSQL connection string |
| `AGENT_SDK_HOST` | `localhost` | Agent SDK server hostname |
| `AGENT_SDK_PORT` | `7233` | Agent SDK gRPC port |
| `AGENT_SDK_NAMESPACE` | `default` | Agent SDK namespace |
| `AGENT_SDK_TASK_QUEUE` | `ai-assistant` | Agent SDK task queue name |
| `LLM_API_KEY` | *(required)* | API key for LLM access |
| `LLM_MODEL` | `gpt-4o` | Model identifier |
| `LLM_BASE_URL` | *(empty)* | LLM base URL with trailing slash (empty uses OpenAI default) |
| `LLM_API_VERSION` | *(empty)* | Azure OpenAI api-version (e.g. `2024-10-01-preview`). When set, uses the Azure client with `api-key` header |
| `SYSTEM_PROMPT` | `You are a helpful assistant.` | System prompt sent to the LLM |
| `CONVERSATION_WINDOW_SIZE` | `20` | Number of recent messages included in LLM context |

Copy the example and fill in your values:

```bash
cp server/.env.example server/.env
# then edit server/.env
```

## API endpoints

| Endpoint | Method | Request | Response |
|----------|--------|---------|----------|
| `/api/conversations` | GET | — | `Conversation[]` |
| `/api/conversations` | POST | `{ title?: string }` | `{ id, title, createdAt }` |
| `/api/conversations/:id` | PATCH | `{ title: string }` | `204 No Content` |
| `/api/conversations/:id` | DELETE | — | `204 No Content` |
| `/api/conversations/:id/messages` | GET | — | `Message[]` |
| `/api/conversations/:id/messages` | POST | `{ content: string }` | `Message` (assistant reply) |

## Database

PostgreSQL is provisioned automatically by Docker Compose. The server runs migrations on startup, creating the `conversations` and `messages` tables if they don't exist.

## Built with

- **[agent-sdk-go](https://github.com/vvsynapse/agent-sdk-go)** — AI agent SDK for Go (powered by [Temporal](https://temporal.io))
- **[chi](https://github.com/go-chi/chi)** — HTTP router
- **[pgx](https://github.com/jackc/pgx)** — PostgreSQL driver
- **PostgreSQL** — Data persistence

---

[← Back to repository README](../README.md)
