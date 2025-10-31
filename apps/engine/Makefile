SHELL := /usr/bin/env bash

APP_NAME := iac-engine
API_CMD := ./cmd/api
WORKER_CMD := ./cmd/worker
MIGRATE_CMD := ./cmd/migrate

GO_FLAGS := -trimpath
LDFLAGS := -s -w

.PHONY: all build api worker test lint dev migrate run-api run-worker tidy deps clean

all: build

deps:
	go mod tidy

build:
	go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/api $(API_CMD)
	go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/worker $(WORKER_CMD)
	go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/migrate $(MIGRATE_CMD)

api:
	go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/api $(API_CMD)

worker:
	go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/worker $(WORKER_CMD)

run-api:
	go run $(API_CMD)

run-worker:
	go run $(WORKER_CMD)

migrate:
	go run $(MIGRATE_CMD)

test:
	go test ./...

lint:
	go vet ./...

dev: 
	$(MAKE) -j2 run-api run-worker

tidy:
	go mod tidy

clean:
	rm -rf bin

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v
