# agent-chat Server

Go API server powered by [agent-sdk-go](https://github.com/agenticenv/agent-sdk-go) and [Temporal](https://temporal.io). Provides REST endpoints for managing conversations and messages, with durable workflow-orchestrated AI agent execution.

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
# Use this when running the UI locally with npm run dev (point SERVER_API_URL at http://localhost:8081)
docker compose up -d postgres temporal server
```

**Ports (host):** The API is published at **`http://localhost:8081`** (Compose maps **8081 → 8080** in the server container). The UI in Docker is at **[http://localhost:3000](http://localhost:3000)**. The browser loads the UI, then calls the API using **`SERVER_API_URL`** baked into `config.json` (not through the UI container as a proxy). **Temporal UI** at **[http://localhost:8233](http://localhost:8233)** is optional: workflow execution visibility, tracing, and debugging — not required to use Agent Chat.

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

### Start Agent Chat

From the **repository root** (where `docker-compose.yml` lives), after you change **server** code or `server/Dockerfile`, rebuild and recreate only the API service (Postgres and Temporal keep running):

```bash
docker compose up -d --build server
```

To rebuild and start the **full** Agent Chat stack: `docker compose up -d --build`.

Optional shortcuts if you use the repo **Makefile**: `make restart-server`, `make up`, `make help`.

## Environment variables

**Docker Compose:** `server` gets Postgres connection (`POSTGRES_*`) and Temporal client settings (`TEMPORAL_*`) from `docker-compose.yml`. The process listens on **8080** inside the container (`config.HTTPListenPort`); Compose maps **host port 8081 → 8080** so a service already using host **8080** does not conflict. **`server/.env`** holds LLM and Agent Chat tuning (`LLM_*`, `AGENT_*`).

**Local `go run` (no Compose):** export `POSTGRES_*`, `TEMPORAL_*`, or set `DATABASE_URL`. Defaults assume `localhost` for Postgres when building the URL from `POSTGRES_*`.

| Variable | Description |
|----------|-------------|
| `LLM_API_KEY` | *(required)* API key for the LLM |
| `LLM_PROVIDER` | e.g. `openai` (default `openai`) |
| `LLM_MODEL` | Model id (default `gpt-4o`) |
| `LLM_BASE_URL` | Optional custom API base |
| `AGENT_SYSTEM_PROMPT` | How Agent Chat should behave — role, tone, and rules for replies (default: short helpful-assistant prompt) |
| `AGENT_NAME` | Display name for the agent (default `agent-chat`) |
| `AGENT_DESCRIPTION` | Short description (default set in code) |
| `AGENT_CONVERSATION_WINDOW_SIZE` | How many recent messages feed into context (default `20`) |
| `LOG_LEVEL` | `debug` \| `info` \| `warn` \| `error` (default `info`) |

Infra (when not using Compose, or to override): `DATABASE_URL` (optional full URL), `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `TEMPORAL_HOST`, `TEMPORAL_PORT`, `TEMPORAL_NAMESPACE`, and related Temporal settings in `config.go` / Compose.

Copy the example and add your LLM key:

```bash
cp server/.env.example server/.env
# then edit server/.env
```

## API endpoints

| Endpoint | Method | Request | Response |
|----------|--------|---------|----------|
| `/api/conversations` | GET | — | `Conversation[]` |
| `/api/conversations` | POST | `{ title?: string }` (empty title defaults to **New chat**) | `{ id, title, createdAt }` |
| `/api/conversations/:id` | PATCH | `{ title: string }` | `204 No Content` |
| `/api/conversations/:id` | DELETE | — | `204 No Content` |
| `/api/conversations/:id/messages` | GET | — | `Message[]` |
| `/api/conversations/:id/messages` | POST | `{ content: string }` | `Message` (assistant reply) |

## Database

PostgreSQL is provisioned automatically by Docker Compose. The server runs migrations on startup, creating the `conversations` and `messages` tables if they don't exist.

## Built with

- **[agent-sdk-go](https://github.com/agenticenv/agent-sdk-go)** — AI agent SDK for Go (powered by [Temporal](https://temporal.io))
- **[chi](https://github.com/go-chi/chi)** — HTTP router
- **[pgx](https://github.com/jackc/pgx)** — PostgreSQL driver
- **PostgreSQL** — Data persistence

---

[← Back to repository README](../README.md)
