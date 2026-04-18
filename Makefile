# Run from this directory (where docker-compose.yml lives).

.PHONY: help up down restart-server restart-ui logs logs-ui secrets-scan

help:
	@echo "Stack:  make up | make down"
	@echo "Rebuild one service (image + container):  make restart-server | make restart-ui"
	@echo "Logs:   make logs  (server)  |  make logs-ui"
	@echo "Other:  make secrets-scan"

up:
	docker compose up -d --build

down:
	docker compose down

# Rebuild image (if needed) and recreate container — use after code/Dockerfile changes.
restart-server:
	docker compose up -d --build server

restart-ui:
	docker compose up -d --build ui

logs:
	docker compose logs -f server

logs-ui:
	docker compose logs -f ui

# Requires: gitleaks (https://github.com/gitleaks/gitleaks) or Docker for zricethezav/gitleaks
secrets-scan:
	@if command -v gitleaks >/dev/null 2>&1; then \
		gitleaks detect --source . --verbose --redact; \
	elif command -v docker >/dev/null 2>&1; then \
		docker run --rm -v "$$(pwd):/repo" -w /repo zricethezav/gitleaks:latest detect --source=/repo --verbose --redact; \
	else \
		echo "Install gitleaks (https://github.com/gitleaks/gitleaks#installing) or Docker."; \
		exit 1; \
	fi
