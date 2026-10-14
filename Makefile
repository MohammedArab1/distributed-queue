-include .env
export

.PHONY: build test lint

build:
	go build -o bin/ ./cmd/...

test:
	go test -race ./...

lint:
	go vet $$(go list ./... | grep -v /examples/)

.PHONY: up down reset-db psql redis-cli

up:
	docker compose up -d --wait

down:
	docker compose down

reset-db:
	docker compose down -v && docker compose up -d --wait

psql:
	docker compose exec postgres psql -U jobqueue jobqueue

redis-cli:
	docker compose exec redis redis-cli
