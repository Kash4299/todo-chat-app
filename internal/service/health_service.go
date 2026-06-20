package service

import (
	"context"
	"database/sql"
)

// pinger is the minimal surface a readiness check needs from a database handle.
// Keeping it an interface lets the service be unit-tested without a real DB.
type pinger interface {
	PingContext(ctx context.Context) error
}

type IHealthService interface {
	// Ready reports whether this instance can serve a request right now.
	Ready(ctx context.Context) error
}

type HealthService struct {
	db pinger
}

func NewHealthService(db *sql.DB) IHealthService {
	return &HealthService{db: db}
}

func (s *HealthService) Ready(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
