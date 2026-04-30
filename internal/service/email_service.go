package service

import (
	"fmt"
	"strings"

	"github.com/Kash4299/todo-chat-app/pkg/email"
)

type IEmailService interface {
	SendVerificationEmail(toEmail, rawToken string) error
}

type SMTPEmailService struct {
	host     string
	port     string
	user     string
	password string
	from     string
	baseURL  string
}

func NewEmailService(cfg *email.SMTPConfig) IEmailService {
	return &SMTPEmailService{
		host:     cfg.Host,
		port:     cfg.Port,
		user:     cfg.User,
		password: cfg.Password,
		from:     cfg.From,
		baseURL:  cfg.BaseURL,
	}
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
