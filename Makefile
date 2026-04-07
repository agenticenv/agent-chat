# Run from this directory (where docker-compose.yml lives).

.PHONY: help secrets-scan up up-build build build-server build-ui restart-server restart-ui logs logs-ui down

help:
	@echo "Targets:"
	@echo "  make secrets-scan    - scan repo for leaked secrets (gitleaks; Docker fallback)"
	@echo "  make build           - docker compose build server ui"
	@echo "  make build-server    - build only the server image"
	@echo "  make build-ui        - build only the ui image"
	@echo "  make restart-server  - rebuild + recreate only server (Postgres/Temporal keep running)"
	@echo "  make restart-ui      - rebuild + recreate only ui"
	@echo "  make up              - docker compose up -d --build (full stack)"
	@echo "  make logs            - follow server logs"
	@echo "  make logs-ui         - follow ui logs"
	@echo "  make down            - docker compose down"
	@echo ""
	@echo "After UI or server code changes: make restart-ui  OR  make restart-server"

build:
	docker compose build server ui

build-server:
	docker compose build server

build-ui:
	docker compose build ui

restart-server:
	docker compose up -d --build server

restart-ui:
	docker compose up -d --build ui

# Default: rebuild when Dockerfile or build context changed, then start everything.
up: up-build

up-build:
	docker compose up -d --build

logs:
	docker compose logs -f server

logs-ui:
	docker compose logs -f ui

down:
	docker compose down
	
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
