package service_test

import (
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/service"
)

func TestNewEmailService_LogTransport_NoSMTPNeeded(t *testing.T) {
	cfg := &config.Config{EmailTransport: "log", AppBaseURL: "https://app.example.com"}
	svc, err := service.NewEmailService(cfg)
	if err != nil {
		t.Fatalf("log transport must not error at boot, got %v", err)
	}
	if svc == nil {
		t.Fatal("expected email service")
	}
	// Log transport must not dial SMTP — sending succeeds with no mail server.
	if err := svc.SendVerificationEmail("a@example.com", "tok"); err != nil {
		t.Fatalf("log transport send must not error, got %v", err)
	}
	if err := svc.SendWorkspaceInvitationEmail("a@example.com", "tok"); err != nil {
		t.Fatalf("log transport send must not error, got %v", err)
	}
}

func TestNewEmailService_DefaultsToLog(t *testing.T) {
	svc, err := service.NewEmailService(&config.Config{}) // empty transport
	if err != nil {
		t.Fatalf("empty transport should default to log (no boot error), got %v", err)
	}
	if svc == nil {
		t.Fatal("expected email service")
	}
}

func TestNewEmailService_InvalidTransport(t *testing.T) {
	if _, err := service.NewEmailService(&config.Config{EmailTransport: "carrier-pigeon"}); err == nil {
		t.Fatal("expected error for invalid EMAIL_TRANSPORT")
	}
}

func TestNewEmailService_SMTPTransport_RequiresConfig(t *testing.T) {
	// smtp mode with missing SMTP_* must error (delegates to NewSMTPConfig),
	// not silently degrade to a no-op.
	if _, err := service.NewEmailService(&config.Config{EmailTransport: "smtp"}); err == nil {
		t.Fatal("expected error when EMAIL_TRANSPORT=smtp but SMTP config missing")
	}
}
