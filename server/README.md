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
# Start all dependencies and the server
docker compose up -d postgres temporal server
```

The server will be available at [http://localhost:8080](http://localhost:8080). Temporal UI is at [http://localhost:8233](http://localhost:8233).

To start everything (server, UI, and all dependencies):

```bash
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
| `LLM_BASE_URL` | *(empty)* | LLM base URL (empty uses provider default) |
| `SYSTEM_PROMPT` | `You are a helpful assistant.` | System prompt sent to the LLM |
| `CONVERSATION_WINDOW_SIZE` | `20` | Number of recent messages included in LLM context |

Set `LLM_API_KEY` in `server/.env` or pass it directly:

```bash
LLM_API_KEY=your-key docker compose up -d server
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

- **[temporal-agent-sdk-go](https://github.com/vvsynapse/temporal-agent-sdk-go)** — AI agent SDK for Go (powered by [Temporal](https://temporal.io))
- **[chi](https://github.com/go-chi/chi)** — HTTP router
- **[pgx](https://github.com/jackc/pgx)** — PostgreSQL driver
- **PostgreSQL** — Data persistence

---

[← Back to repository README](../README.md)
