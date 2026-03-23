package core

import "fmt"

// ValidateServerConfig checks the server config and returns errors (blocking)
// and warnings (non-blocking). Errors prevent sending; warnings are informational
// (the frontend shows them via a confirmation dialog before calling Send).
func ValidateServerConfig(cfg ServerConfig) (errs []string, warns []string) {
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
		if !cfg.TLS && !cfg.SSL {
			errs = append(errs, "AUTH requires TLS or SSL — plaintext AUTH is not supported")
		}
	}

	return errs, warns
}

// ValidateMailConfig checks the mail config and returns errors (blocking)
// and warnings (non-blocking). Email format is NOT validated — the user needs to
// test how mail servers handle invalid or empty addresses.
func ValidateMailConfig(cfg MailConfig) (errs []string, warns []string) {
	if cfg.MailNumber < 1 || cfg.MailNumber > 100000 {
		errs = append(errs, fmt.Sprintf("Mail Number must be 1–100,000, got %d", cfg.MailNumber))
	}

	if cfg.ThreadNumber < 1 {
		errs = append(errs, fmt.Sprintf("Thread Number must be at least 1, got %d", cfg.ThreadNumber))
	}

	if cfg.IntervalMs < 0 || cfg.IntervalMs > 60000 {
		errs = append(errs, fmt.Sprintf("Interval must be 0–60,000 ms, got %d", cfg.IntervalMs))
	}

	if cfg.ContentType != "" && cfg.ContentType != "text/plain" && cfg.ContentType != "text/html" {
		errs = append(errs, "Content-Type must be text/plain or text/html")
	}

	if cfg.MailNumber >= 100 {
		warns = append(warns, fmt.Sprintf("Sending %d mails — please confirm", cfg.MailNumber))
	}

	if cfg.ThreadNumber > 50 {
		warns = append(warns, fmt.Sprintf("Thread count %d exceeds 50 — may overload the server", cfg.ThreadNumber))
	}

	return errs, warns
}
