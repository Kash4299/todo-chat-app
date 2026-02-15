# Simple Makefile for a Go project

# Build the application
all: build test

include .env
export $(shell sed 's/=.*//' .env)

build:
	@echo "Building..."
	
	
	@go build -o main cmd/main.go

# Run the application
dev:
	@go run cmd/main.go

migration-dev:
	@migrate -database postgresql://$(DB_USERNAME):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_DATABASE)?sslmode=disable -path migrations -verbose up

migration-dev-down:
	@migrate -database postgresql://$(DB_USERNAME):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_DATABASE)?sslmode=disable -path migrations -verbose down $(num)

# Create DB container
docker-run:
	@if docker compose up --build 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up --build; \
	fi

# Shutdown DB container
docker-down:
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Test the application
test:
	@echo "Testing..."
	@go test ./internal/services/... ./internal/grpc/... -v -count=1

# Test with coverage report
test-cover:
	@echo "Running tests with coverage..."
	@go test ./internal/services/... ./internal/grpc/... -v -coverprofile=coverage.out
	@go tool cover -func=coverage.out
	@echo "To view HTML report: go tool cover -html=coverage.out"
# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

# Generate Go code from proto files
proto-gen:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto
	@echo "Protobuf code generated successfully"

.PHONY: all build run test clean watch docker-run docker-down itest proto-gen
