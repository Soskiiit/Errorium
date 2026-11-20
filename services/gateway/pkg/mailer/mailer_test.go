package mailer_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	"github.com/soskiiit/errorium/services/gateway/pkg/mailer"
)

func TestLogMailer_SendReset_WritesToBuffer(t *testing.T) {
	var buf bytes.Buffer
	m := mailer.NewLogMailer(&buf)

	to := "testuser@example.com"
	link := "http://localhost:3000/reset-password?token=xyz123"

	err := m.SendReset(context.Background(), to, link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, to) {
		t.Errorf("expected buffer to contain recipient %q, got: %s", to, out)
	}
	if !strings.Contains(out, link) {
		t.Errorf("expected buffer to contain link %q, got: %s", link, out)
	}
}

func TestLogMailer_SendReset_CancelledContext(t *testing.T) {
	var buf bytes.Buffer
	m := mailer.NewLogMailer(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.SendReset(ctx, "user@example.com", "http://localhost:3000/reset-password?token=123")
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected buffer to be empty on cancelled context, got %q", buf.String())
	}
}

func TestNewMailer_Factory(t *testing.T) {
	// Empty host returns LogMailer
	devMailer := mailer.New(config.SMTPConfig{Host: ""})
	if _, ok := devMailer.(*mailer.LogMailer); !ok {
		t.Errorf("expected *mailer.LogMailer for empty host, got %T", devMailer)
	}

	// Non-empty host returns SMTPMailer
	smtpMailer := mailer.New(config.SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "no-reply@example.com",
	})
	if _, ok := smtpMailer.(*mailer.SMTPMailer); !ok {
		t.Errorf("expected *mailer.SMTPMailer for configured host, got %T", smtpMailer)
	}
}

func TestSMTPMailer_ConstructorAndConfigErrors(t *testing.T) {
	cfg := config.SMTPConfig{
		Host:     "127.0.0.1",
		Port:     1, // unreachable port
		User:     "user",
		Password: "password",
		From:     "no-reply@errorium.ru",
	}

	m := mailer.NewSMTPMailer(cfg)
	if m == nil {
		t.Fatal("expected non-nil SMTPMailer")
	}

	// Calling SendReset to an unreachable address should fail with an error
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := m.SendReset(ctx, "recipient@example.com", "http://localhost:3000/reset-password?token=tok")
	if err == nil {
		t.Fatal("expected error connecting to unreachable port, got nil")
	}
}

func TestSMTPMailer_CancelledContext(t *testing.T) {
	cfg := config.SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
	}
	m := mailer.NewSMTPMailer(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.SendReset(ctx, "recipient@example.com", "http://localhost:3000/reset-password?token=tok")
	if err == nil {
		t.Fatal("expected error with cancelled context, got nil")
	}
}
