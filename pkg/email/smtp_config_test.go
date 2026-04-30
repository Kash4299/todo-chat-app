package email_test

import (
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/pkg/email"
)

func TestNewSMTPConfig_RejectsMissingConfig(t *testing.T) {
	_, err := email.NewSMTPConfig(&config.Config{})
	if err == nil {
		t.Fatal("expected missing SMTP config error")
	}
}

func TestNewSMTPConfig_RejectsInvalidAppBaseURL(t *testing.T) {
	_, err := email.NewSMTPConfig(&config.Config{
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

func TestNewSMTPConfig_FailsWhenSMTPConnectionFails(t *testing.T) {
	_, err := email.NewSMTPConfig(&config.Config{
		SMTPHost:     "127.0.0.1",
		SMTPPort:     "1",
		SMTPUser:     "user",
		SMTPPassword: "password",
		SMTPFrom:     "noreply@example.com",
		AppBaseURL:   "https://app.example.com",
	})
	if err == nil {
		t.Fatal("expected SMTP connection error")
	}
}
