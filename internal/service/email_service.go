package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
)

const smtpTimeout = 10 * time.Second

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

func NewEmailService(cfg *config.Config) (IEmailService, error) {
	host := strings.TrimSpace(cfg.SMTPHost)
	port := strings.TrimSpace(cfg.SMTPPort)
	user := strings.TrimSpace(cfg.SMTPUser)
	password := strings.TrimSpace(cfg.SMTPPassword)
	from := strings.TrimSpace(cfg.SMTPFrom)
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.AppBaseURL), "/")

	if host == "" || port == "" || user == "" || password == "" || from == "" {
		return nil, fmt.Errorf("SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, and SMTP_FROM are required")
	}
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil || parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return nil, fmt.Errorf("APP_BASE_URL must be an absolute http or https URL")
	}
	if parsedBaseURL.Scheme != "http" && parsedBaseURL.Scheme != "https" {
		return nil, fmt.Errorf("APP_BASE_URL must use http or https")
	}

	return &SMTPEmailService{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
		baseURL:  baseURL,
	}, nil
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

	addr := s.host + ":" + s.port
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	tlsCfg := &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: smtpTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(smtpTimeout))

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return err
	}
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
