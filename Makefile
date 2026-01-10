run-orchestrator:
	go run ./cmd/orchestrator

run-media:
	go run ./cmd/media

run-quota:
	go run ./cmd/quota

run-ingest:
	go run ./cmd/ingest

run-processing:
	go run ./cmd/processing

run-publish:
	go run ./cmd/publish

build:
	go build ./cmd/...
