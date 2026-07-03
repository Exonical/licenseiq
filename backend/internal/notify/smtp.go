package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/Exonical/licenseiq/backend/internal/config"
)

type SMTPNotifier struct {
	cfg config.SMTPNotificationsConfig
}

func NewSMTPNotifier(cfg config.SMTPNotificationsConfig) (*SMTPNotifier, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.From) == "" || len(cfg.Recipients) == 0 {
		return nil, fmt.Errorf("smtp configuration is incomplete")
	}
	switch cfg.TLSMode {
	case "", "off", "starttls", "implicit":
	default:
		return nil, fmt.Errorf("unsupported smtp tls mode %q", cfg.TLSMode)
	}
	if cfg.Port <= 0 {
		cfg.Port = 587
	}
	return &SMTPNotifier{cfg: cfg}, nil
}

func (n *SMTPNotifier) Name() string { return "smtp" }

func (n *SMTPNotifier) Send(ctx context.Context, msg Message) error {
	if n == nil {
		return nil
	}
	payload, err := buildSMTPMessage(n.cfg.From, n.cfg.Recipients, msg)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(n.cfg.Host, fmt.Sprintf("%d", n.cfg.Port))
	var conn net.Conn
	switch n.cfg.TLSMode {
	case "implicit":
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: n.cfg.Host})
	default:
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	}
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, n.cfg.Host)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if n.cfg.TLSMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: n.cfg.Host}); err != nil {
				return err
			}
		}
	}
	if n.cfg.Username != "" || n.cfg.Password != "" {
		if err := client.Auth(smtp.PlainAuth("", n.cfg.Username, n.cfg.Password, n.cfg.Host)); err != nil {
			return err
		}
	}
	if err := client.Mail(n.cfg.From); err != nil {
		return err
	}
	for _, rcpt := range n.cfg.Recipients {
		if err := client.Rcpt(strings.TrimSpace(rcpt)); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func buildSMTPMessage(from string, recipients []string, msg Message) ([]byte, error) {
	var buf bytes.Buffer
	headers := map[string]string{
		"From":         from,
		"To":           strings.Join(recipients, ", "),
		"Subject":      msg.Subject,
		"MIME-Version": "1.0",
	}
	if msg.HTML != "" {
		headers["Content-Type"] = `multipart/alternative; boundary="licenseiq-boundary"`
		for k, v := range headers {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
		buf.WriteString("\r\n")
		buf.WriteString("--licenseiq-boundary\r\n")
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		buf.WriteString(msg.Text)
		buf.WriteString("\r\n--licenseiq-boundary\r\n")
		buf.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		buf.WriteString(msg.HTML)
		buf.WriteString("\r\n--licenseiq-boundary--\r\n")
		return buf.Bytes(), nil
	}
	headers["Content-Type"] = "text/plain; charset=utf-8"
	for k, v := range headers {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
	}
	buf.WriteString("\r\n")
	buf.WriteString(msg.Text)
	buf.WriteString("\r\n")
	return buf.Bytes(), nil
}
