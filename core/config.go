package core

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	appName        = "dMailSender"
	keyringService = "dMailSender"
	keyringUser    = "smtp_auth_credentials"
)

// ConfigDir returns the platform config directory for the app.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}

// ConfigPath returns the full path to config.json.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig reads config.json from the platform config directory.
// If the file does not exist, returns DefaultConfig.
func LoadConfig() (AppConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	return cfg, nil
}

// LoadConfigFrom reads config from an arbitrary file path.
func LoadConfigFrom(path string) (AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), err
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	return cfg, nil
}

// SaveConfig writes config.json to the platform config directory.
func SaveConfig(cfg AppConfig) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// marshalConfig serializes an AppConfig to JSON with indentation.
func marshalConfig(cfg AppConfig) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
}

// unmarshalConfig deserializes JSON bytes into an AppConfig.
func unmarshalConfig(data []byte) (AppConfig, error) {
	var cfg AppConfig
	err := json.Unmarshal(data, &cfg)
	return cfg, err
}

// smtpCredentials is the JSON structure stored in the OS keychain.
type smtpCredentials struct {
	ID string `json:"id"`
	PW string `json:"pw"`
}

// SavePassword stores auth ID and password as JSON in the OS keychain under a fixed key.
func SavePassword(authID, password string) error {
	data, err := json.Marshal(smtpCredentials{ID: authID, PW: password})
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, keyringUser, string(data))
}

// LoadPassword retrieves auth ID and password from the OS keychain.
func LoadPassword() (authID, password string, err error) {
	raw, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return "", "", err
	}
	var cred smtpCredentials
	if err := json.Unmarshal([]byte(raw), &cred); err != nil {
		return "", "", err
	}
	return cred.ID, cred.PW, nil
}

// DeletePassword removes the stored credentials from the OS keychain.
func DeletePassword() error {
	return keyring.Delete(keyringService, keyringUser)
}

// MigrateKeychain migrates credentials from the old per-authID key to the new fixed key.
// Called once at startup. If the new key already exists, migration is skipped.
func MigrateKeychain(oldAuthID string) {
	if oldAuthID == "" {
		return
	}
	// Already migrated?
	if _, _, err := LoadPassword(); err == nil {
		return
	}
	// Try loading from old key
	pw, err := keyring.Get(keyringService, oldAuthID)
	if err != nil || pw == "" {
		return
	}
	// Save to new key
	_ = SavePassword(oldAuthID, pw)
	// Clean up old key
	_ = keyring.Delete(keyringService, oldAuthID)
}
