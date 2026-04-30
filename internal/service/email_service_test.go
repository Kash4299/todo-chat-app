package service_test

import (
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/Kash4299/todo-chat-app/pkg/email"
)

func TestNewEmailService_AcceptsSMTPConfig(t *testing.T) {
	svc := service.NewEmailService(&email.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     "465",
		User:     "user",
		Password: "password",
		From:     "noreply@example.com",
		BaseURL:  "https://app.example.com",
	})
	if svc == nil {
		t.Fatal("expected email service")
	}
}
