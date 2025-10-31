FROM golang:1.25-alpine AS base

WORKDIR /app

RUN apk add --no-cache git build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Default command is overridden by docker-compose
CMD ["go", "run", "./cmd/api"]


