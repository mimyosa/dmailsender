package core

import "fmt"

// ValidateServerConfig checks the server config and returns a list of error messages.
// Only checks required fields and range constraints.
func ValidateServerConfig(cfg ServerConfig) []string {
	var errs []string

	if cfg.SMTP == "" {
		errs = append(errs, "SMTP host is required")
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		errs = append(errs, fmt.Sprintf("Port must be 1–65535, got %d", cfg.Port))
	}

	if cfg.TLS && cfg.SSL {
		errs = append(errs, "TLS and SSL cannot both be enabled")
	}

	if cfg.Auth {
		if cfg.AuthID == "" {
			errs = append(errs, "Auth ID is required when auth is enabled")
		}
	}

	return errs
}

// ValidateMailConfig checks the mail config and returns a list of error messages.
// Only checks range constraints. Email format is NOT validated — the user needs to
// test how mail servers handle invalid or empty addresses.
func ValidateMailConfig(cfg MailConfig) []string {
	var errs []string

	if cfg.MailNumber < 1 || cfg.MailNumber > 100000 {
		errs = append(errs, fmt.Sprintf("Mail Number must be 1–100,000, got %d", cfg.MailNumber))
	}

	if cfg.ThreadNumber < 1 || cfg.ThreadNumber > 50 {
		errs = append(errs, fmt.Sprintf("Thread Number must be 1–50, got %d", cfg.ThreadNumber))
	}

	if cfg.IntervalMs < 0 || cfg.IntervalMs > 60000 {
		errs = append(errs, fmt.Sprintf("Interval must be 0–60,000 ms, got %d", cfg.IntervalMs))
	}

	if cfg.ContentType != "" && cfg.ContentType != "text/plain" && cfg.ContentType != "text/html" {
		errs = append(errs, "Content-Type must be text/plain or text/html")
	}

	return errs
}
