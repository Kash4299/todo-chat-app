package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg := config.NewConfig()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode,
	)

	log.Println("Running database migrations...")

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	// You could support flags here (up, down, version) but we default to "up"
	if len(os.Args) > 1 && os.Args[1] == "down" {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run migrate down: %v", err)
		}
		log.Println("Database migrations (down) completed successfully!")
		return
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migrate up: %v", err)
	}

	log.Println("Database migrations (up) completed successfully!")
}
