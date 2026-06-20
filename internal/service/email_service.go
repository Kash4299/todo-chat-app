package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/pkg/email"
)

type IEmailService interface {
	SendVerificationEmail(toEmail, rawToken string) error
	SendWorkspaceInvitationEmail(toEmail, rawToken string) error
}

type SMTPEmailService struct {
	host     string
	port     string
	user     string
	password string
	from     string
	baseURL  string
}

// NewEmailService selects an email transport based on EMAIL_TRANSPORT.
//
// "smtp" builds (and connection-checks) a real SMTP transport — email is a
// critical dependency in that mode, so a misconfiguration fails startup.
// "log" (the default) is a degradable transport, like Redis/Kafka: it writes
// verification and invitation links to the application log instead of sending
// mail, so the app boots and auth flows work without any SMTP server (dev / v0).
func NewEmailService(cfg *config.Config) (IEmailService, error) {
	transport := strings.ToLower(strings.TrimSpace(cfg.EmailTransport))
	switch transport {
	case "smtp":
		smtpCfg, err := email.NewSMTPConfig(cfg)
		if err != nil {
			return nil, err
		}
		return &SMTPEmailService{
			host:     smtpCfg.Host,
			port:     smtpCfg.Port,
			user:     smtpCfg.User,
			password: smtpCfg.Password,
			from:     smtpCfg.From,
			baseURL:  smtpCfg.BaseURL,
		}, nil
	case "", "log":
		log.Printf("WARNING: EMAIL_TRANSPORT=%q — verification/invitation links are written to logs, not emailed. Set EMAIL_TRANSPORT=smtp in production.", transport)
		return &LogEmailService{baseURL: strings.TrimRight(strings.TrimSpace(cfg.AppBaseURL), "/")}, nil
	default:
		return nil, fmt.Errorf("invalid EMAIL_TRANSPORT %q (want \"smtp\" or \"log\")", transport)
	}
}

// LogEmailService is a degradable email transport for development and the first
// deploy: it logs the action link instead of sending an email.
type LogEmailService struct {
	baseURL string
}

func (s *LogEmailService) SendVerificationEmail(toEmail, rawToken string) error {
	log.Printf("[email:log] verification link for %s -> %s", toEmail, s.link("/verify-email", rawToken))
	return nil
}

func (s *LogEmailService) SendWorkspaceInvitationEmail(toEmail, rawToken string) error {
	log.Printf("[email:log] workspace invitation link for %s -> %s", toEmail, s.link("/workspace/accept-invitation", rawToken))
	return nil
}

func (s *LogEmailService) link(path, rawToken string) string {
	if s.baseURL == "" {
		return fmt.Sprintf("%s?token=%s", path, rawToken)
	}
	return fmt.Sprintf("%s%s?token=%s", s.baseURL, path, rawToken)
}

func (s *SMTPEmailService) SendVerificationEmail(toEmail, rawToken string) error {
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", strings.TrimRight(s.baseURL, "/"), rawToken)

	subject := "Verify your email"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:560px;margin:40px auto;padding:0 16px">
  <h2>Verify your email address</h2>
  <p>Click the button below to verify your email. The link expires in <strong>24 hours</strong>.</p>
  <a href="%s" style="display:inline-block;padding:12px 24px;background:#4f46e5;color:#fff;text-decoration:none;border-radius:6px">Verify email</a>
  <p style="margin-top:24px;color:#6b7280;font-size:13px">If you did not create an account, you can safely ignore this email.</p>
</body>
</html>`, verifyURL)

	msg := buildMIMEMessage(s.from, toEmail, subject, body)

	client, err := email.NewAuthenticatedSMTPClient(&email.SMTPConfig{
		Host:     s.host,
		Port:     s.port,
		User:     s.user,
		Password: s.password,
		From:     s.from,
		BaseURL:  s.baseURL,
	})
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Mail(s.from); err != nil {
		return err
	}
	if err := client.Rcpt(toEmail); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = fmt.Fprint(w, msg); err != nil {
		return err
	}
	return w.Close()
}

func buildMIMEMessage(from, to, subject, htmlBody string) string {
	return fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, htmlBody,
	)
}

func (s *SMTPEmailService) SendWorkspaceInvitationEmail(toEmail, rawToken string) error {
	inviteURL := fmt.Sprintf("%s/workspace/accept-invitation?token=%s", strings.TrimRight(s.baseURL, "/"), rawToken)

	subject := "You've been invited to a workspace"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:560px;margin:40px auto;padding:0 16px">
  <h2>You've been invited to join a workspace</h2>
  <p>Click the button below to accept the invitation. The link expires in <strong>48 hours</strong>.</p>
  <a href="%s" style="display:inline-block;padding:12px 24px;background:#4f46e5;color:#fff;text-decoration:none;border-radius:6px">Accept invitation</a>
  <p style="margin-top:24px;color:#6b7280;font-size:13px">If you were not expecting this invitation, you can safely ignore this email.</p>
</body>
</html>`, inviteURL)

	msg := buildMIMEMessage(s.from, toEmail, subject, body)

	client, err := email.NewAuthenticatedSMTPClient(&email.SMTPConfig{
		Host:     s.host,
		Port:     s.port,
		User:     s.user,
		Password: s.password,
		From:     s.from,
		BaseURL:  s.baseURL,
	})
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Mail(s.from); err != nil {
		return err
	}
	if err := client.Rcpt(toEmail); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = fmt.Fprint(w, msg); err != nil {
		return err
	}
	return w.Close()
}
