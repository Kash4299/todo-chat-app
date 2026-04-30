package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

func NewAuthenticatedSMTPClient(cfg *SMTPConfig) (*smtp.Client, error) {
	addr := cfg.Host + ":" + cfg.Port
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	tlsCfg := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	dialer := &net.Dialer{Timeout: smtpTimeout}

	var (
		client *smtp.Client
		err    error
	)

	switch cfg.Port {
	case "465":
		conn, dialErr := tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
		if dialErr != nil {
			return nil, dialErr
		}
		_ = conn.SetDeadline(time.Now().Add(smtpTimeout))
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
	default:
		conn, dialErr := dialer.Dial("tcp", addr)
		if dialErr != nil {
			return nil, dialErr
		}
		_ = conn.SetDeadline(time.Now().Add(smtpTimeout))
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}

		ok, _ := client.Extension("STARTTLS")
		if !ok {
			_ = client.Close()
			return nil, fmt.Errorf("smtp server does not support STARTTLS on port %s", cfg.Port)
		}
		if err := client.StartTLS(tlsCfg); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	if err := client.Auth(auth); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}
