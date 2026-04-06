# Agent Chat

Sample chat app built with [agent-sdk-go](https://github.com/agenticenv/agent-sdk-go): React UI, Go REST API, and durable workflow-backed conversations.

## Prerequisites

- **Docker** — [Docker Engine](https://docs.docker.com/engine/) with **Docker Compose** (the `docker compose` CLI; Compose v2 is bundled with Docker Desktop and current Engine installs).
- **LLM access** — An API key from a supported provider (for example OpenAI or an OpenAI-compatible HTTP API). Add it to **`server/.env`** in the **Configuration** section below.
- **This repository** — Clone or copy the Agent Chat project so you have the **`docker-compose.yml`** at the repo root.

Local UI development (`npm run dev`) needs **Node.js** — see **[ui/README.md](ui/README.md)** if you run the UI outside Docker.

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

- **[ui/README.md](ui/README.md)** — local UI dev (`npm run dev`), `SERVER_API_URL`, Docker image, rebuild the UI with Docker Compose.
- **[server/README.md](server/README.md)** — environment variables, architecture, REST API, rebuild the API with Docker Compose.

## Stack

- **[agent-sdk-go](https://github.com/agenticenv/agent-sdk-go)** — AI agent SDK for Go
- **React Router 7 + Vite** — UI
- **Tailwind CSS v4** — Styling
- **react-markdown** + **remark-gfm** — Message bubbles render Markdown (GFM)
