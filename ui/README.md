# agent-chat UI

React Router 7 + Vite + Tailwind CSS.

- **Local dev (`npm run dev`)**: The browser uses same-origin `/api`; Vite proxies `/api` to **`SERVER_API_URL`**.
- **Production (`react-router-serve` in Docker)**: The Node server does **not** proxy `/api`. **`entrypoint.sh`** writes `config.json` with `apiBase` (**`${SERVER_API_URL}/api`**) and `enableStream` (from **`ENABLE_STREAM`**) so the browser calls the API directly and picks streaming vs REST (URLs must be **reachable from the browser**, e.g. `http://localhost:9090` when Compose maps **9090 → 8080** (browser uses host **9090**; the server process still listens on **8080** inside its container). The Go API enables CORS for that.
- **Messages**: Chat content is rendered as **Markdown** (GitHub-flavored via `remark-gfm`): lists, code fences, tables, links, etc. Plain text is valid Markdown.
- **Streaming**: By default the UI uses SSE streaming — tokens appear as the agent generates them. In Docker, set **`ENABLE_STREAM=false`** on the `ui` service (runtime — no rebuild); in local dev use **`VITE_ENABLE_STREAM=false`** in `ui/.env`.

## Run locally

```bash
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173). Start the Go API (or point `SERVER_API_URL` at it). Copy `.env.example` to `.env` if you need a non-default backend URL.

## Environment variables

| Variable | Where | Description |
|----------|-------|-------------|
| `SERVER_API_URL` | `.env` (dev) / Docker | Backend **origin** only, no `/api` suffix — must be reachable from the **browser** (not a Docker-only hostname like `http://server:8080`). **Vite** uses it for the dev proxy (`/api` → `${SERVER_API_URL}/api`). **Docker** `entrypoint.sh` bakes the same value into `config.json` for client-side `fetch`. Compose defaults to **`http://localhost:9090`** (host **9090** → container **8080**, so local **8080** stays free). |
| `VITE_ENABLE_STREAM` | `ui/.env` (local dev only) | `true` (default) = SSE streaming; `false` = REST. Baked in at Vite dev server startup. |
| `ENABLE_STREAM` | Docker `ui` service env / `docker run -e` | Same meaning as `VITE_ENABLE_STREAM`, but read at **container start** — `entrypoint.sh` writes `enableStream` into `config.json`. No image rebuild. Compose default **`true`**. |
| `PORT` | Docker | HTTP port for `react-router-serve`. Default `3000`. |

## Streaming vs REST mode

The UI supports two response modes:

| Mode | Value | Behavior |
|------|-------|----------|
| **Streaming** (default) | `true` or unset | Uses `POST .../messages/stream` — tokens appear incrementally as the agent generates them via SSE |
| **REST** | `false` | Uses `POST .../messages` — waits for the agent to finish, then displays the full response at once |

Both modes show a typing indicator while waiting. The backend supports both endpoints — no server changes needed.

**Local dev:** set `VITE_ENABLE_STREAM=false` (or `true`) in `ui/.env` and restart `npm run dev`.

**Docker:** set `ENABLE_STREAM` on the `ui` service (see `docker-compose.yml`) or pass `-e ENABLE_STREAM=false`. The entrypoint regenerates `config.json` on each start — **restart the container**, no rebuild:

```bash
ENABLE_STREAM=false docker compose up -d ui
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
  -e SERVER_API_URL=http://localhost:9090 \
  -e PORT=3000 \
  agent-chat-ui
```

With **docker compose**, `server` maps **host 9090 → container 8080**, and `ui` sets **`SERVER_API_URL`** (default `http://localhost:9090`, see `docker-compose.yml`). Override the host mapping or `SERVER_API_URL` if needed. Rebuild the UI image after changing `entrypoint.sh`.

### Start Agent Chat

From the **repository root** (where `docker-compose.yml` lives), after you change **UI** code or the UI Dockerfile, rebuild and recreate only the UI service (Postgres and Temporal keep running):

```bash
docker compose up -d --build ui
```

To rebuild and start the **full** Agent Chat stack: `docker compose up -d --build`.

Optional shortcuts if you use the repo **Makefile**: run `make help`, or e.g. `make up`, `make restart-ui`, `make logs-ui`, `make secrets-scan`.

---

[← Back to repository README](../README.md)
