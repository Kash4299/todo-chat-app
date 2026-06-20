package service

import (
	"context"
	"errors"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) PingContext(ctx context.Context) error { return f.err }

func TestHealthService_Ready_OK(t *testing.T) {
	s := &HealthService{db: fakePinger{err: nil}}
	if err := s.Ready(context.Background()); err != nil {
		t.Fatalf("expected nil when DB ping succeeds, got %v", err)
	}
}

func TestHealthService_Ready_DBDown(t *testing.T) {
	s := &HealthService{db: fakePinger{err: errors.New("db down")}}
	if err := s.Ready(context.Background()); err == nil {
		t.Fatal("expected error when DB ping fails")
	}
}
