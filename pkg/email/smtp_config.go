package email

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
)

const smtpTimeout = 10 * time.Second

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	BaseURL  string
}

func NewSMTPConfig(cfg *config.Config) (*SMTPConfig, error) {
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

	smtpCfg := &SMTPConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		From:     from,
		BaseURL:  baseURL,
	}

	if err := verifySMTPConnection(smtpCfg); err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	return smtpCfg, nil
}

func verifySMTPConnection(cfg *SMTPConfig) error {
	client, err := NewAuthenticatedSMTPClient(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Quit()
}
