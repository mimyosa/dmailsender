package core

import (
	"encoding/json"
	"testing"
)

func TestAppConfigJSONRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.SMTP = "smtp.example.com"
	cfg.Server.Port = 587
	cfg.Server.TLS = true
	cfg.Server.Auth = true
	cfg.Server.AuthID = "user@example.com"
	cfg.Mail.MailFrom = "sender@example.com"
	cfg.Mail.RcptTo = "recipient@example.com"
	cfg.Mail.Subject = "Test Subject"
	cfg.Mail.Body = "Hello World"
	cfg.Mail.CustomHeaders = []Header{
		{Key: "X-Custom", Value: "test"},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var restored AppConfig
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if restored.Server.SMTP != cfg.Server.SMTP {
		t.Errorf("SMTP mismatch: got %q, want %q", restored.Server.SMTP, cfg.Server.SMTP)
	}
	if restored.Server.Port != cfg.Server.Port {
		t.Errorf("Port mismatch: got %d, want %d", restored.Server.Port, cfg.Server.Port)
	}
	if restored.Mail.MailFrom != cfg.Mail.MailFrom {
		t.Errorf("MailFrom mismatch: got %q, want %q", restored.Mail.MailFrom, cfg.Mail.MailFrom)
	}
	if len(restored.Mail.CustomHeaders) != 1 || restored.Mail.CustomHeaders[0].Key != "X-Custom" {
		t.Errorf("CustomHeaders mismatch: got %v", restored.Mail.CustomHeaders)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.SMTP != "localhost" {
		t.Errorf("default SMTP: got %q, want 'localhost'", cfg.Server.SMTP)
	}
	if cfg.Server.Port != 25 {
		t.Errorf("default Port: got %d, want 25", cfg.Server.Port)
	}
	if cfg.Mail.ContentType != "text/plain" {
		t.Errorf("default ContentType: got %q, want 'text/plain'", cfg.Mail.ContentType)
	}
	if cfg.Mail.MailNumber != 1 {
		t.Errorf("default MailNumber: got %d, want 1", cfg.Mail.MailNumber)
	}
	if cfg.Mail.ThreadNumber != 1 {
		t.Errorf("default ThreadNumber: got %d, want 1", cfg.Mail.ThreadNumber)
	}
}
