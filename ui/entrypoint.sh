#!/bin/sh
# Inject API base URL from env at container start (runtime config)
# Default: /api (same-origin)
API_BASE="${API_BASE:-/api}"
printf '{"apiBase":"%s"}\n' "$API_BASE" > /app/build/client/config.json
# Remove static index.html so react-router-serve's SSR handler handles "/"
# instead of express.static intercepting it before SSR can inject scripts.
rm -f /app/build/client/index.html
exec "$@"
