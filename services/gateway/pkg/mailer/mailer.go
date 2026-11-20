// Package mailer provides email sending capabilities for password resets and notifications.
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/smtp"
	"strings"

	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

// Mailer abstracts sending emails (such as password reset links).
type Mailer interface {
	SendReset(ctx context.Context, to, link string) error
}

// LogMailer logs password reset links instead of sending real emails (for dev and testing).
type LogMailer struct {
	writer io.Writer
}

var _ Mailer = (*LogMailer)(nil)

// NewLogMailer constructs a LogMailer, optionally writing to a provided writer.
func NewLogMailer(out ...io.Writer) *LogMailer {
	var w io.Writer
	if len(out) > 0 {
		w = out[0]
	}
	return &LogMailer{writer: w}
}

// SendReset logs the password reset link.
func (m *LogMailer) SendReset(ctx context.Context, to, link string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	msg := fmt.Sprintf("[DEV MAILER] password reset link for %s: %s\n", to, link)
	if m.writer != nil {
		_, err := io.WriteString(m.writer, msg)
		return err
	}

	log.Print(msg)
	return nil
}

// SMTPMailer sends emails via SMTP with TLS (port 465) or STARTTLS.
type SMTPMailer struct {
	cfg config.SMTPConfig
}

var _ Mailer = (*SMTPMailer)(nil)

// NewSMTPMailer constructs a new SMTPMailer.
func NewSMTPMailer(cfg config.SMTPConfig) *SMTPMailer {
	if cfg.Port <= 0 {
		cfg.Port = 587
	}
	if strings.TrimSpace(cfg.From) == "" {
		cfg.From = "no-reply@errorium.ru"
	}
	return &SMTPMailer{cfg: cfg}
}

// SendReset sends a password reset email via SMTP.
func (m *SMTPMailer) SendReset(ctx context.Context, to, link string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(m.cfg.Host) == "" {
		return fmt.Errorf("smtp host is not configured")
	}

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	from := m.cfg.From
	if from == "" {
		from = "no-reply@errorium.ru"
	}

	subject := "Errorium: сброс пароля"
	body := fmt.Sprintf("Здравствуйте!\r\n\r\nДля сброса пароля перейдите по ссылке:\r\n%s\r\n\r\nСсылка действительна в течение 1 часа.\r\nЕсли вы не запрашивали сброс пароля, проигнорируйте это письмо.\r\n", link)

	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body))

	var dialer net.Dialer
	var conn net.Conn
	var err error

	if m.cfg.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: m.cfg.Host,
		}
		rawConn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		tlsConn := tls.Client(rawConn, tlsConfig)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = rawConn.Close()
			return fmt.Errorf("smtp tls handshake: %w", err)
		}
		conn = tlsConn
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("smtp dial: %w", err)
		}
	}

	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Close()

	if m.cfg.Port != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName: m.cfg.Host,
			}
			if err := c.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	if m.cfg.User != "" || m.cfg.Password != "" {
		if ok, _ := c.Extension("AUTH"); ok {
			auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write msg: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return c.Quit()
}

// New returns a Mailer instance based on the configuration.
// If Host is empty, it returns a LogMailer and outputs a warning log.
func New(cfg config.SMTPConfig) Mailer {
	if strings.TrimSpace(cfg.Host) == "" {
		log.Println("[WARN] SMTP_HOST is not configured; using LogMailer (dev mode)")
		return NewLogMailer()
	}
	return NewSMTPMailer(cfg)
}
