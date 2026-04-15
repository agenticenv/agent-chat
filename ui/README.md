# agent-chat UI

React Router 7 + Vite + Tailwind CSS.

- **Local dev (`npm run dev`)**: The browser uses same-origin `/api`; Vite proxies `/api` to **`SERVER_API_URL`**.
- **Production (`react-router-serve` in Docker)**: The Node server does **not** proxy `/api`. **`entrypoint.sh`** writes `config.json` with `apiBase` = **`${SERVER_API_URL}/api`** so the browser calls the API directly (must be a URL **reachable from the browser**, e.g. `http://localhost:8081` when Compose maps the API to host port **8081**). The Go API enables CORS for that.
- **Messages**: Chat content is rendered as **Markdown** (GitHub-flavored via `remark-gfm`): lists, code fences, tables, links, etc. Plain text is valid Markdown.
- **Streaming**: By default the UI uses SSE streaming — tokens appear as the agent generates them. Set `VITE_STREAMING=false` to switch to REST mode (full response appears at once).

## Run locally

```bash
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173). Start the Go API (or point `SERVER_API_URL` at it). Copy `.env.example` to `.env` if you need a non-default backend URL.

## Environment variables

| Variable | Where | Description |
|----------|-------|-------------|
| `SERVER_API_URL` | `.env` (dev) / Docker | Backend **origin** only, no `/api` suffix — must be reachable from the **browser** (not a Docker-only hostname like `http://server:8080`). **Vite** uses it for the dev proxy (`/api` → `${SERVER_API_URL}/api`). **Docker** `entrypoint.sh` bakes the same value into `config.json` for client-side `fetch`. Compose defaults to **`http://localhost:8081`** (API published on host **8081** to avoid clashing with a local **8080**). |
| `VITE_STREAMING` | `.env` (dev) / Docker build arg | `true` (default) = SSE streaming mode; `false` = REST mode. See **Streaming vs REST mode** below. |
| `PORT` | Docker | HTTP port for `react-router-serve`. Default `3000`. |

## Streaming vs REST mode

The UI supports two response modes, controlled by `VITE_STREAMING` in `ui/.env`:

| Mode | Value | Behavior |
|------|-------|----------|
| **Streaming** (default) | `true` or unset | Uses `POST .../messages/stream` — tokens appear incrementally as the agent generates them via SSE |
| **REST** | `false` | Uses `POST .../messages` — waits for the agent to finish, then displays the full response at once |

Both modes show a typing indicator while waiting. The backend supports both endpoints — no server changes needed.

**To switch modes (local dev):**
1. Set `VITE_STREAMING=false` (or `true`) in `ui/.env`
2. Restart `npm run dev` — Vite bakes this value in at startup

**To switch modes (Docker):**

`VITE_STREAMING` is baked in at image build time. Rebuild the UI image after changing it:

```bash
# ui/.env must contain VITE_STREAMING=false before building
docker compose up -d --build ui
```

### Local dev — backend on a different host/port

```bash
# .env next to package.json
SERVER_API_URL=http://localhost:3001
```

Restart `npm run dev` after changing.

## Docker

Build from the **repository root**:

```bash
docker build -t agent-chat-ui ./ui
```

Run (API must be reachable from the **browser** at `SERVER_API_URL`):

```bash
docker run -p 3000:3000 \
  -e SERVER_API_URL=http://localhost:8081 \
  -e PORT=3000 \
  agent-chat-ui
```

With **docker compose**, `server` maps **host 8081 → container 8080**, and `ui` sets **`SERVER_API_URL`** (default `http://localhost:8081`, see `docker-compose.yml`). Override the host mapping or `SERVER_API_URL` if needed. Rebuild the UI image after changing `entrypoint.sh`.

### Start Agent Chat

From the **repository root** (where `docker-compose.yml` lives), after you change **UI** code or the UI Dockerfile, rebuild and recreate only the UI service (Postgres and Temporal keep running):

```bash
docker compose up -d --build ui
```

To rebuild and start the **full** Agent Chat stack: `docker compose up -d --build`.

Optional shortcuts if you use the repo **Makefile**: `make restart-ui`, `make up`, `make help`.

---

[← Back to repository README](../README.md)
