package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSaveLoad(t *testing.T) {
	// Use a temp dir instead of real config dir
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.Server.SMTP = "mail.test.com"
	cfg.Server.Port = 465
	cfg.Server.SSL = true
	cfg.Mail.MailFrom = "test@test.com"
	cfg.Mail.RcptTo = "dest@test.com"
	cfg.Mail.MailNumber = 10
	cfg.Mail.ThreadNumber = 5

	// Save
	data, err := marshalConfig(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Load
	readData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	loaded, err := unmarshalConfig(readData)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if loaded.Server.SMTP != "mail.test.com" {
		t.Errorf("SMTP: got %q, want 'mail.test.com'", loaded.Server.SMTP)
	}
	if loaded.Server.Port != 465 {
		t.Errorf("Port: got %d, want 465", loaded.Server.Port)
	}
	if !loaded.Server.SSL {
		t.Errorf("SSL should be true")
	}
	if loaded.Mail.MailNumber != 10 {
		t.Errorf("MailNumber: got %d, want 10", loaded.Mail.MailNumber)
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir error: %v", err)
	}
	if dir == "" {
		t.Fatal("ConfigDir returned empty string")
	}
	if filepath.Base(dir) != appName {
		t.Errorf("ConfigDir base: got %q, want %q", filepath.Base(dir), appName)
	}
}
