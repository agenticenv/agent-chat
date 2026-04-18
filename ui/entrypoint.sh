#!/bin/sh
# SERVER_API_URL = backend origin (no /api), as reachable from the browser.
# - npm run dev: Vite proxies /api -> ${SERVER_API_URL}/api (see vite.config.ts).
# - Docker (react-router-serve): no /api proxy — write config.json so the browser calls ${SERVER_API_URL}/api.
# Default matches docker-compose (API published on host :9090 → container :8080); override via compose or -e.
SERVER_API_URL="${SERVER_API_URL:-http://localhost:9090}"
export SERVER_API_URL
ORIG="${SERVER_API_URL%/}"

# SSE streaming vs REST: same as VITE_ENABLE_STREAM in local dev (runtime here, not build-time).
ENABLE_STREAM="${ENABLE_STREAM:-true}"
export ENABLE_STREAM
case "$(printf '%s' "$ENABLE_STREAM" | tr '[:upper:]' '[:lower:]')" in
  false|0|no|off) STREAM_JSON=false ;;
  *) STREAM_JSON=true ;;
esac

printf '{"apiBase":"%s/api","enableStream":%s}\n' "$ORIG" "$STREAM_JSON" > /app/build/client/config.json
rm -f /app/build/client/index.html
exec "$@"
