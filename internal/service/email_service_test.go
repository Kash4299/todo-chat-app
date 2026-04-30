package service_test

import (
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/service"
)

func TestNewEmailService_RejectsMissingConfig(t *testing.T) {
	_, err := service.NewEmailService(&config.Config{})
	if err == nil {
		t.Fatal("expected missing SMTP config error")
	}
}

func TestNewEmailService_RejectsInvalidAppBaseURL(t *testing.T) {
	_, err := service.NewEmailService(&config.Config{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "465",
		SMTPUser:     "user",
		SMTPPassword: "password",
		SMTPFrom:     "noreply@example.com",
		AppBaseURL:   "localhost:3000",
	})
	if err == nil {
		t.Fatal("expected invalid APP_BASE_URL error")
	}
}

func TestNewEmailService_AcceptsValidConfig(t *testing.T) {
	svc, err := service.NewEmailService(&config.Config{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "465",
		SMTPUser:     "user",
		SMTPPassword: "password",
		SMTPFrom:     "noreply@example.com",
		AppBaseURL:   "https://app.example.com",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if svc == nil {
		t.Fatal("expected email service")
	}
}
