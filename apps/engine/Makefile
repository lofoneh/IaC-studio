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
	cd apps/engine && go mod tidy

build:
	cd apps/engine && go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/api $(API_CMD)
	cd apps/engine && go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/worker $(WORKER_CMD)
	cd apps/engine && go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/migrate $(MIGRATE_CMD)

api:
	cd apps/engine && go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/api $(API_CMD)

worker:
	cd apps/engine && go build $(GO_FLAGS) -ldflags "$(LDFLAGS)" -o bin/worker $(WORKER_CMD)

run-api:
	cd apps/engine && go run $(API_CMD)

run-worker:
	cd apps/engine && go run $(WORKER_CMD)

migrate:
	cd apps/engine && go run $(MIGRATE_CMD)

test:
	cd apps/engine && go test ./...

lint:
	cd apps/engine && go vet ./...

dev: 
	$(MAKE) -j2 run-api run-worker

tidy:
	cd apps/engine && go mod tidy

clean:
	rm -rf apps/engine/bin
