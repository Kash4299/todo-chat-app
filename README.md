# Todo API

A Go-based REST API for managing todo items with support for Redis caching and Kafka event streaming.

## Features

- RESTful API for todo management
- PostgreSQL database with GORM
- Redis caching (optional)
- Kafka event streaming (optional)
- Graceful shutdown
- Dependency injection with Uber FX
- Structured logging with Zap

## Environment Configuration

### Core Configuration

```bash
# Server Configuration
PORT=8080
APP_ENV=local

# Database
DB_HOST=localhost
DB_PORT=5432
DB_DATABASE=postgres
DB_USERNAME=postgres
DB_PASSWORD=123456
DB_SCHEMA=public
DB_NAME=postgres
```

### Redis Configuration

To enable Redis caching, set the following environment variables:

```bash
# Enable Redis (true/false)
ENABLE_REDIS=true

# Redis Connection
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Kafka Configuration

To enable Kafka event streaming, set the following environment variables:

```bash
# Enable Kafka (true/false)
ENABLE_KAFKA=true

# Kafka Connection
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=todo-events
```

## Running the Application

### Prerequisites

- Go 1.24+
- PostgreSQL (if using database)
- Redis (if ENABLE_REDIS=true)
- Kafka (if ENABLE_KAFKA=true)

### Development

```bash
# Run with default configuration (Redis and Kafka disabled)
go run ./cmd/main.go

# Run with Redis enabled
ENABLE_REDIS=true go run ./cmd/main.go

# Run with Kafka enabled
ENABLE_KAFKA=true go run ./cmd/main.go

# Run with both Redis and Kafka enabled
ENABLE_REDIS=true ENABLE_KAFKA=true go run ./cmd/main.go
```

### Production

```bash
# Build the application
go build -o todo-api ./cmd/main.go

# Run with environment variables
ENABLE_REDIS=true ENABLE_KAFKA=true ./todo-api
```

## API Endpoints

- `GET /health` - Health check
- `GET /api/v1/todos` - Get all todos
- `GET /api/v1/todos/:id` - Get todo by ID
- `POST /api/v1/todos` - Create new todo
- `PUT /api/v1/todos/:id` - Update todo
- `DELETE /api/v1/todos/:id` - Delete todo

## Architecture

The application uses a clean architecture with:

- **Handlers**: HTTP request/response handling
- **Services**: Business logic with Redis caching and Kafka events
- **Repositories**: Data access layer
- **Database**: PostgreSQL with GORM
- **Redis**: Optional caching layer
- **Kafka**: Optional event streaming

## Graceful Shutdown

The application implements graceful shutdown for all components:

1. Database connection
2. Redis connection (if enabled)
3. Kafka connections (if enabled)
4. HTTP server

## Dependencies

- **Web Framework**: Gin
- **Database**: GORM + PostgreSQL
- **Caching**: Redis (optional)
- **Event Streaming**: Kafka (optional)
- **Dependency Injection**: Uber FX
- **Logging**: Zap
- **Configuration**: Environment variables 