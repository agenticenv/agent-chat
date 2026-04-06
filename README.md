# Agent Chat

A demo app showcasing [agent-sdk-go](https://github.com/agenticenv/agent-sdk-go) — the Temporal-first AI agent SDK for Go. Built with a React UI and Go REST API.backed conversations.

> This is a demo app showcasing [agent-sdk-go](https://github.com/agenticenv/agent-sdk-go). Not intended for production use.

## Why agent-sdk-go

Most agent frameworks run in-process — if your server restarts, the agent run is lost. [agent-sdk-go](https://github.com/agenticenv/agent-sdk-go) is Temporal-first, so every agent run is a durable workflow:

- **Durable conversations** — chat history and agent runs survive server restarts
- **Long-running agents** — conversations can run for extended periods without losing state
- **Automatic retries** — failed LLM calls retry automatically via Temporal

## Stack

- **[agent-sdk-go](https://github.com/agenticenv/agent-sdk-go)** — AI agent SDK for Go
- **React Router 7 + Vite** — UI
- **Tailwind CSS v4** — Styling
- **react-markdown** + **remark-gfm** — Message bubbles render Markdown (GFM)

## Prerequisites

- **Docker** — [Docker Engine](https://docs.docker.com/engine/) with **Docker Compose** (the `docker compose` CLI; Compose v2 is bundled with Docker Desktop and current Engine installs).
- **LLM access** — An API key from a supported provider (for example OpenAI or an OpenAI-compatible HTTP API). Add it to **`server/.env`** in the **Configuration** section below.
- **This repository** — Clone or copy the Agent Chat project so you have the **`docker-compose.yml`** at the repo root.

## How to start

Agent Chat runs with **Docker Compose**. Run every command below from the **repository root** — the directory that contains **`docker-compose.yml`**.

### Configuration (Required)

Agent Chat reads **`server/.env`** for LLM settings. If **`LLM_API_KEY`** is missing, **the Agent Chat API will not start** and containers may fail or restart.

- **Copy** the example file:

  ```bash
  cp server/.env.example server/.env
  ```

- **Required**

  | Variable | You must… |
  |----------|-----------|
  | **`LLM_API_KEY`** | Set to your real LLM API key. An empty placeholder means Agent Chat cannot start. |

- **Optional — LLM** (defaults are fine for OpenAI)

  - **`LLM_PROVIDER`** — default `openai`
  - **`LLM_MODEL`** — default `gpt-4o`
  - **`LLM_BASE_URL`** — set only for a **custom or Azure-style** HTTP endpoint; use a **`LLM_MODEL`** your provider supports

- **Optional — agent**

  - **`AGENT_SYSTEM_PROMPT`** — how Agent Chat behaves (role, tone, rules). Omit to use the built-in default.
  - **`AGENT_NAME`**, **`AGENT_DESCRIPTION`**, **`AGENT_CONVERSATION_WINDOW_SIZE`** — labeling and how much chat history is in context; see **`server/.env.example`**.

Full variable list and behavior: **[server/README.md](server/README.md)**.

### Start (Docker Compose)

- **Start the stack** (Postgres, Temporal, API, UI):

  ```bash
  docker compose up -d --build
  ```

- **Open Agent Chat:** **[http://localhost:3000](http://localhost:3000)** — use the chat in your browser.

- **(Optional)** **Temporal UI:** **[http://localhost:8233](http://localhost:8233)** — view Temporal workflow executions for Agent Chat.

### Stop (Docker Compose)

```bash
docker compose down
```

## References

- **[ui/README.md](ui/README.md)** — `SERVER_API_URL`, Docker image, rebuilding the UI with Docker Compose.
- **[server/README.md](server/README.md)** — environment variables, architecture, REST API, rebuild the API with Docker Compose.