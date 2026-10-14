.PHONY: up down db api web install seed health

up: ## Start Postgres (Docker) + remind about Ollama
	docker compose up -d
	@echo "Postgres on :5433 — start Ollama with: ollama serve"

down:
	docker compose down

db:
	docker compose up -d db

install:
	cd apps/web && npm install
	cd apps/api && go mod tidy

api:
	cd apps/api && go run ./cmd/server

web:
	cd apps/web && npm run dev

health:
	curl -s http://localhost:8080/api/health | python3 -m json.tool
