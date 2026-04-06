# Run from this directory (where docker-compose.yml lives).

.PHONY: help up up-build build build-server build-ui restart-server restart-ui logs logs-ui down

help:
	@echo "Targets:"
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
