.PHONY: help infra infra-down infra-logs db-init db-reset build \
        run-orchestrator run-ingest run-processing run-publish run-media \
        run-all stop-all status clean

COMPOSE_FILE = deploy/docker-compose.yml
DATABASE_URL  = postgres://mediauser:mediapass@localhost:5433/mediadb?sslmode=disable
SQL_FILE      = sql/script.sql

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ── Infrastructure ───────────────────────────────────────────────

infra: ## Start Kafka + Postgres (DB auto-inits from sql/script.sql)
	docker compose -f $(COMPOSE_FILE) up -d
	@echo "Waiting for Postgres..."
	@until docker compose -f $(COMPOSE_FILE) exec -T postgres pg_isready -U mediauser -d mediadb > /dev/null 2>&1; do sleep 1; done
	@echo "Waiting for Kafka..."
	@until docker compose -f $(COMPOSE_FILE) exec -T kafka kafka-topics --bootstrap-server localhost:9092 --list > /dev/null 2>&1; do sleep 1; done
	@echo "Infrastructure is ready."

infra-down: ## Stop infrastructure
	docker compose -f $(COMPOSE_FILE) down

infra-logs: ## Tail infrastructure logs
	docker compose -f $(COMPOSE_FILE) logs -f

# ── Database ─────────────────────────────────────────────────────

db-init: ## Apply schema (sql/script.sql) to running Postgres
	psql "$(DATABASE_URL)" -f $(SQL_FILE)

db-reset: ## Drop all tables and re-apply schema
	psql "$(DATABASE_URL)" -c "\
		DROP TABLE IF EXISTS orchestrator_outbox, sagas, \
		ingest_outbox, upload_sessions, \
		processing_outbox, processing_tasks, \
		publish_outbox, publications, \
		media CASCADE;"
	psql "$(DATABASE_URL)" -f $(SQL_FILE)
	@echo "Database reset complete."

# ── Build ────────────────────────────────────────────────────────

build: ## Build all services into bin/
	@mkdir -p bin
	go build -o bin/orchestrator ./cmd/orchestrator
	go build -o bin/ingest      ./cmd/ingest
	go build -o bin/processing  ./cmd/processing
	go build -o bin/publish     ./cmd/publish
	go build -o bin/media       ./cmd/media
	@echo "All binaries in bin/"

# ── Run (foreground, one service) ────────────────────────────────

run-orchestrator: ## Run orchestrator (foreground)
	go run ./cmd/orchestrator

run-ingest: ## Run ingest (foreground)
	go run ./cmd/ingest

run-processing: ## Run processing (foreground)
	go run ./cmd/processing

run-publish: ## Run publish (foreground)
	go run ./cmd/publish

run-media: ## Run media (foreground)
	go run ./cmd/media

# ── Run all (background) ────────────────────────────────────────

run-all: build ## Build & run all services in background
	@mkdir -p .pids
	@echo "Starting services..."
	@bin/orchestrator & echo $$! > .pids/orchestrator.pid
	@bin/ingest       & echo $$! > .pids/ingest.pid
	@bin/processing   & echo $$! > .pids/processing.pid
	@bin/publish      & echo $$! > .pids/publish.pid
	@bin/media        & echo $$! > .pids/media.pid
	@echo "All services started. PIDs in .pids/"

stop-all: ## Stop all background services
	@for f in .pids/*.pid; do \
		if [ -f "$$f" ]; then \
			pid=$$(cat "$$f"); \
			name=$$(basename "$$f" .pid); \
			if kill -0 "$$pid" 2>/dev/null; then \
				kill "$$pid" && echo "Stopped $$name ($$pid)"; \
			else \
				echo "$$name already stopped"; \
			fi; \
			rm -f "$$f"; \
		fi; \
	done

# ── Status ───────────────────────────────────────────────────────

status: ## Check infrastructure + services health
	@echo "=== Docker ==="
	@docker compose -f $(COMPOSE_FILE) ps --format "table {{.Name}}\t{{.Status}}" 2>/dev/null || echo "  docker compose not running"
	@echo ""
	@echo "=== Services ==="
	@for svc in orchestrator:8084 ingest:8082 publish:8083 media:8081; do \
		name=$${svc%%:*}; port=$${svc##*:}; \
		if curl -sf http://localhost:$$port/health > /dev/null 2>&1; then \
			echo "  $$name ($$port) ✓"; \
		else \
			echo "  $$name ($$port) ✗"; \
		fi; \
	done
	@echo "  processing      (no HTTP)"

# ── Cleanup ──────────────────────────────────────────────────────

clean: ## Remove binaries, PIDs, uploaded/output files
	rm -rf bin .pids uploads output published
