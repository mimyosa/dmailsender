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

// SavePassword stores a password in the OS keychain.
func SavePassword(authID, password string) error {
	return keyring.Set(keyringService, authID, password)
}

// LoadPassword retrieves a password from the OS keychain.
func LoadPassword(authID string) (string, error) {
	return keyring.Get(keyringService, authID)
}

// DeletePassword removes a password from the OS keychain.
func DeletePassword(authID string) error {
	return keyring.Delete(keyringService, authID)
}
