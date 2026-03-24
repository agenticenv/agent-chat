#!/bin/sh
# Inject API base URL from env at container start (runtime config)
# Default: /api (same-origin)
API_BASE="${API_BASE:-/api}"
printf '{"apiBase":"%s"}\n' "$API_BASE" > /app/build/client/config.json
exec "$@"
