package core

import "testing"

func TestValidateServerConfig(t *testing.T) {
	// Valid config
	cfg := ServerConfig{SMTP: "smtp.example.com", Port: 587, TLS: true}
	if errs, _ := ValidateServerConfig(cfg); len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}

	// Empty SMTP
	cfg = ServerConfig{SMTP: "", Port: 587}
	if errs, _ := ValidateServerConfig(cfg); len(errs) == 0 {
		t.Error("expected error for empty SMTP")
	}

	// Invalid port
	cfg = ServerConfig{SMTP: "smtp.example.com", Port: 0}
	if errs, _ := ValidateServerConfig(cfg); len(errs) == 0 {
		t.Error("expected error for port 0")
	}

	cfg = ServerConfig{SMTP: "smtp.example.com", Port: 70000}
	if errs, _ := ValidateServerConfig(cfg); len(errs) == 0 {
		t.Error("expected error for port 70000")
	}

	// Both TLS and SSL
	cfg = ServerConfig{SMTP: "smtp.example.com", Port: 587, TLS: true, SSL: true}
	if errs, _ := ValidateServerConfig(cfg); len(errs) == 0 {
		t.Error("expected error for both TLS and SSL")
	}

	// Auth without ID
	cfg = ServerConfig{SMTP: "smtp.example.com", Port: 587, Auth: true, AuthID: ""}
	if errs, _ := ValidateServerConfig(cfg); len(errs) == 0 {
		t.Error("expected error for auth without ID")
	}

	// Auth without TLS/SSL — should be error (plaintext AUTH not supported)
	cfg = ServerConfig{SMTP: "smtp.example.com", Port: 587, Auth: true, AuthID: "user"}
	if errs, _ := ValidateServerConfig(cfg); len(errs) == 0 {
		t.Error("expected error for auth without TLS/SSL")
	}
}

func TestValidateMailConfig(t *testing.T) {
	// Valid config
	cfg := MailConfig{
		MailFrom:     "sender@example.com",
		RcptTo:       "recipient@example.com",
		ContentType:  "text/plain",
		MailNumber:   10,
		ThreadNumber: 2,
		IntervalMs:   100,
	}
	if errs, _ := ValidateMailConfig(cfg); len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}

	// Empty email should be allowed (no format validation)
	cfg.MailFrom = ""
	cfg.RcptTo = ""
	if errs, _ := ValidateMailConfig(cfg); len(errs) != 0 {
		t.Errorf("expected no errors for empty emails, got %v", errs)
	}

	// Invalid email format should be allowed
	cfg.MailFrom = "not-an-email"
	cfg.RcptTo = "also-invalid"
	if errs, _ := ValidateMailConfig(cfg); len(errs) != 0 {
		t.Errorf("expected no errors for invalid email format, got %v", errs)
	}
	cfg.MailFrom = "sender@example.com"
	cfg.RcptTo = "recipient@example.com"

	// Mail number out of range
	cfg.MailNumber = 0
	if errs, _ := ValidateMailConfig(cfg); len(errs) == 0 {
		t.Error("expected error for mail number 0")
	}
	cfg.MailNumber = 100001
	if errs, _ := ValidateMailConfig(cfg); len(errs) == 0 {
		t.Error("expected error for mail number > 100000")
	}
	cfg.MailNumber = 10

	// Thread number out of range
	cfg.ThreadNumber = 0
	if errs, _ := ValidateMailConfig(cfg); len(errs) == 0 {
		t.Error("expected error for thread number 0")
	}
	cfg.ThreadNumber = 2

	// Thread > 50 — should be warning, not error
	cfg.ThreadNumber = 51
	errs, warns := ValidateMailConfig(cfg)
	if len(errs) != 0 {
		t.Errorf("expected no errors for thread 51, got %v", errs)
	}
	if len(warns) == 0 {
		t.Error("expected warning for thread number > 50")
	}
	cfg.ThreadNumber = 2

	// MailNumber >= 100 — should be warning
	cfg.MailNumber = 100
	errs, warns = ValidateMailConfig(cfg)
	if len(errs) != 0 {
		t.Errorf("expected no errors for mail number 100, got %v", errs)
	}
	if len(warns) == 0 {
		t.Error("expected warning for mail number >= 100")
	}
	cfg.MailNumber = 10

	// Invalid content type
	cfg.ContentType = "application/json"
	if errs, _ := ValidateMailConfig(cfg); len(errs) == 0 {
		t.Error("expected error for invalid content type")
	}

	// Empty content type should be allowed
	cfg.ContentType = ""
	if errs, _ := ValidateMailConfig(cfg); len(errs) != 0 {
		t.Errorf("expected no errors for empty content type, got %v", errs)
	}
}
