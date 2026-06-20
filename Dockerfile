FROM golang:1.25-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o migrate cmd/migrate/main.go

# Final stage
FROM alpine:3.22

# Install ca-certificates for HTTPS requests and wget for container healthchecks.
RUN apk --no-cache add ca-certificates wget

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binaries and migrations from builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/migrate .
COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 8080

CMD ["./main"]
