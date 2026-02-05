.PHONY: help infra infra-down infra-logs db-init db-reset build \
       run-orchestrator run-ingest run-processing run-publish run-media \
       run-all stop-all status clean

COMPOSE_FILE   = deploy/docker-compose.yml
DATABASE_URL   = postgres://mediauser:mediapass@localhost:5433/mediadb?sslmode=disable
KAFKA_BROKERS  = localhost:9092

# ─── Help ───────────────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ─── Infrastructure ─────────────────────────────────────────────────────────────

infra: ## Start Kafka, PostgreSQL, Redis via Docker Compose
	docker compose -f $(COMPOSE_FILE) up -d
	@echo "Waiting for postgres…"
	@until docker exec mp_postgres pg_isready -U mediauser -d mediadb > /dev/null 2>&1; do sleep 1; done
	@echo "Waiting for kafka…"
	@until docker exec mp_kafka kafka-topics --bootstrap-server localhost:9092 --list > /dev/null 2>&1; do sleep 1; done
	@echo "Infrastructure is ready."

infra-down: ## Stop all infrastructure containers
	docker compose -f $(COMPOSE_FILE) down

infra-logs: ## Tail infrastructure logs
	docker compose -f $(COMPOSE_FILE) logs -f

# ─── Database ────────────────────────────────────────────────────────────────────

db-init: ## Apply SQL schema (runs automatically on first 'make infra')
	PGPASSWORD=mediapass psql -h localhost -p 5433 -U mediauser -d mediadb -f sql/script.sql

db-reset: ## Drop and recreate all tables
	PGPASSWORD=mediapass psql -h localhost -p 5433 -U mediauser -d mediadb -c "\
		DROP TABLE IF EXISTS orchestrator_outbox, sagas, \
		publish_outbox, publications, \
		processing_outbox, processing_tasks, \
		ingest_outbox, assets, \
		media_outbox, quotas CASCADE;"
	$(MAKE) db-init

# ─── Build ───────────────────────────────────────────────────────────────────────

build: ## Build all services into bin/
	@mkdir -p bin
	go build -o bin/orchestrator ./cmd/orchestrator
	go build -o bin/ingest      ./cmd/ingest
	go build -o bin/processing  ./cmd/processing
	go build -o bin/publish     ./cmd/publish
	go build -o bin/media       ./cmd/media
	@echo "All binaries in bin/"

# ─── Run individual services (foreground) ────────────────────────────────────────

run-orchestrator: ## Run orchestrator service
	go run ./cmd/orchestrator

run-ingest: ## Run ingest service
	go run ./cmd/ingest

run-processing: ## Run processing service
	go run ./cmd/processing

run-publish: ## Run publish service
	go run ./cmd/publish

run-media: ## Run media service
	go run ./cmd/media

# ─── Run all services (background) ──────────────────────────────────────────────

run-all: build ## Build & run all services in background
	@mkdir -p .pids logs
	@echo "Starting services…"
	@./bin/orchestrator > logs/orchestrator.log 2>&1 & echo $$! > .pids/orchestrator.pid
	@./bin/ingest      > logs/ingest.log      2>&1 & echo $$! > .pids/ingest.pid
	@./bin/processing  > logs/processing.log  2>&1 & echo $$! > .pids/processing.pid
	@./bin/publish     > logs/publish.log     2>&1 & echo $$! > .pids/publish.pid
	@./bin/media       > logs/media.log       2>&1 & echo $$! > .pids/media.pid
	@echo "All services started. PIDs:"
	@for f in .pids/*.pid; do printf "  %-16s PID %s\n" "$$(basename $$f .pid)" "$$(cat $$f)"; done
	@echo "Logs in logs/  |  Stop with: make stop-all"

stop-all: ## Stop all background services
	@if [ -d .pids ]; then \
		for f in .pids/*.pid; do \
			pid=$$(cat "$$f" 2>/dev/null); \
			name=$$(basename "$$f" .pid); \
			if kill -0 "$$pid" 2>/dev/null; then \
				kill "$$pid" && echo "Stopped $$name ($$pid)"; \
			else \
				echo "$$name already stopped"; \
			fi; \
		done; \
		rm -rf .pids; \
	else \
		echo "No running services found"; \
	fi

# ─── Status ──────────────────────────────────────────────────────────────────────

status: ## Show infrastructure and service status
	@echo "=== Docker containers ==="
	@docker compose -f $(COMPOSE_FILE) ps
	@echo ""
	@echo "=== Services ==="
	@for svc_port in orchestrator:8084 ingest:8081 publish:8083 media:8085; do \
		svc=$${svc_port%%:*}; port=$${svc_port##*:}; \
		if curl -s -o /dev/null -w '' http://localhost:$$port/health 2>/dev/null; then \
			printf "  \033[32m●\033[0m %-16s http://localhost:%s\n" "$$svc" "$$port"; \
		else \
			printf "  \033[31m●\033[0m %-16s (not responding)\n" "$$svc"; \
		fi; \
	done
	@printf "  \033[33m●\033[0m %-16s (no HTTP, kafka-only)\n" "processing"

# ─── Clean ───────────────────────────────────────────────────────────────────────

clean: ## Remove build artifacts, logs, pid files
	rm -rf bin/ logs/ .pids/
