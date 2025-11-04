package server

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type smtpMailer struct {
	cfg    SMTPConfig
	logger *log.Logger
}

func newSMTPMailer(cfg SMTPConfig, logger *log.Logger) mailer {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Port = strings.TrimSpace(cfg.Port)
	cfg.User = strings.TrimSpace(cfg.User)
	cfg.Pass = cfg.Pass
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.Security = strings.ToLower(strings.TrimSpace(cfg.Security))
	if cfg.Security == "" {
		cfg.Security = "starttls"
	}
	if cfg.Host == "" || cfg.From == "" {
		logger.Printf("mailer disabled; SMTP host or from missing")
		return &noopMailer{}
	}
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	logger.Printf("mailer enabled host=%s port=%s security=%s user=%s", cfg.Host, cfg.Port, cfg.Security, maskForLog(cfg.User))
	return &smtpMailer{cfg: cfg, logger: logger}
}

type noopMailer struct{}

func (n *noopMailer) SendResetPassword(string, string, time.Time) error { return nil }
func (n *noopMailer) Enabled() bool                                     { return false }

func (m *smtpMailer) Enabled() bool {
	return true
}

func (m *smtpMailer) SendResetPassword(to, token string, expires time.Time) error {
    link := fmt.Sprintf("http://localhost:5173/reset-password?token=%s", token)
	body := fmt.Sprintf("You requested a password reset. Use the token below before %s UTC.\n\nToken: %s\nReset link: %s\n\nIf you did not request this, ignore the message.",
		expires.UTC().Format(time.RFC3339), token, link)
	msg := message(m.cfg.From, to, "Your VaultCraft password reset link", body)

	switch m.cfg.Security {
	case "ssl", "smtps":
		return m.sendSSL(to, msg)
	case "none":
		return smtp.SendMail(m.addr(), nil, m.cfg.From, []string{to}, msg)
	default:
		return m.sendStartTLS(to, msg)
	}
}

func (m *smtpMailer) sendStartTLS(to string, msg []byte) error {
	addr := m.addr()
	host, _, _ := net.SplitHostPort(addr)

	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		cfg := &tls.Config{ServerName: host}
		if err := client.StartTLS(cfg); err != nil {
			return err
		}
	}

	if m.cfg.User != "" && m.cfg.Pass != "" {
		auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(m.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (m *smtpMailer) sendSSL(to string, msg []byte) error {
	addr := m.addr()
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.Host})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if m.cfg.User != "" && m.cfg.Pass != "" {
		auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(m.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func (m *smtpMailer) addr() string {
	return net.JoinHostPort(m.cfg.Host, m.cfg.Port)
}

func message(from, to, subject, body string) []byte {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	buf.WriteString("\r\n")
	return buf.Bytes()
}

func maskForLog(s string) string {
	if s == "" {
		return "(none)"
	}
	if len(s) <= 2 {
		return "***"
	}
	return s[:1] + "***" + s[len(s)-1:]
}
