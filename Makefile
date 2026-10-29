-include .env
export

.PHONY: build test lint run-server

build:
	go build -o bin/ ./cmd/...

test:
	go test -race ./...

lint:
	go vet $$(go list ./... | grep -v /examples/)

run-server:
	go run ./cmd/server

.PHONY: up down reset-db migrate-up migrate-down migrate-status psql redis-cli

up:
	docker compose up -d --wait

down:
	docker compose down

reset-db:
	docker compose down -v && docker compose up -d --wait
	$(MAKE) migrate-up

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

psql:
	docker compose exec postgres psql -U jobqueue jobqueue

redis-cli:
	docker compose exec redis redis-cli
