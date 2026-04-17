# Agent Chat

A demo app showcasing [agent-sdk-go](https://github.com/agenticenv/agent-sdk-go) — the Temporal-first AI agent SDK for Go. Built with a React UI and Go API, with durable workflow-backed conversations and real-time streaming via SSE.

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
- **A local copy of this project** — You need the files on your computer before you can run anything (clone with Git or download a ZIP — **Get the code** below).

## How to start

Follow these steps in order. Every shell command assumes your **current directory** is the **repository root**: the folder that contains **`docker-compose.yml`**.

### Get the code

- **Clone with Git** (recommended):

  ```bash
  git clone https://github.com/agenticenv/agent-chat.git
  cd agent-chat
  ```

  Use your fork’s URL if you forked the repo. After `cd`, you should see `docker-compose.yml` in that directory.

- **Or download a ZIP** — On GitHub, open **Code** → **Download ZIP**, unzip it, then open a terminal and `cd` into the unzipped folder (the one that contains **`docker-compose.yml`**).

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

- **Copy** the UI example file (optional — only needed to change defaults):

  ```bash
  cp ui/.env.example ui/.env
  ```

- **Optional — UI**

  | Variable | Default | Description |
  |----------|---------|-------------|
  | **`ENABLE_STREAM`** | `true` | `true` = SSE streaming (tokens appear as they're generated); `false` = REST (full response appears at once). See **[ui/README.md](ui/README.md)**. |

Full variable list and behavior: **[ui/README.md](ui/README.md)**.

### Start (Docker Compose)

Agent Chat runs with **Docker Compose** from the repository root.

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

- **[UI](ui/README.md)** — `SERVER_API_URL`, Docker image, rebuilding the UI with Docker Compose.
- **[server](server/README.md)** — environment variables, architecture, REST API, rebuild the API with Docker Compose.