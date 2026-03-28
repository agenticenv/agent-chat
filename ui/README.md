# agent-demo UI

React Router 7 + Vite + Tailwind CSS. Talks to the backend over `/api` (proxied in dev).

## Run locally

```bash
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173). Requests to `/api` are proxied to the backend (see `vite.config.ts`). Start the Go server (or point the proxy at your API) for full functionality.

## Environment variables

| Variable | Where | Description |
|----------|-------|-------------|
| `API_PROXY_TARGET` | `.env` in this folder | Backend URL for the Vite dev proxy. Default `http://localhost:8080`. Set when the API runs on another host/port. |
| `API_BASE` | Docker / runtime | API base URL baked into `config.json` at container start. Default `/api`. Use a full URL when the API is on another origin. |
| `PORT` | Docker | HTTP port for `react-router-serve`. Default `3000`. |

### Local dev — backend on a different port

Create `.env` next to `package.json`:

```bash
API_PROXY_TARGET=http://localhost:3001
```

Restart `npm run dev` after changing.

## Docker

Build the UI image from the **repository root** (parent of `ui/`):

```bash
docker build -t agent-demo-ui ./ui
```

Run:

```bash
docker run -p 3000:3000 -e API_BASE=http://localhost:8080/api -e PORT=3000 agent-demo-ui
```

`API_BASE` is written to `config.json` at container start by `entrypoint.sh`.

With **docker compose** at the repo root, set `API_BASE` under the `ui` service (see `docker-compose.yml`).

---

[← Back to repository README](../README.md)
