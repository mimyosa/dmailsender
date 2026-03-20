package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"dMailSender/core"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppService is the bridge between frontend and core engine.
// It delegates all business logic to core/ and only handles
// data forwarding and Wails event emission.
type AppService struct {
	ctx         context.Context
	config      core.AppConfig
	cancel      context.CancelFunc
	sending     bool
	mu          sync.Mutex
	attachments []string // file paths for Input mode attachments
}

// NewAppService creates a new AppService instance.
func NewAppService() *AppService {
	return &AppService{}
}

// startup is called by Wails on app start.
func (a *AppService) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := core.LoadConfig()
	if err != nil {
		runtime.LogWarningf(ctx, "Failed to load config: %v", err)
	}
	a.config = cfg
}

// beforeClose is called by Wails before the app closes.
func (a *AppService) beforeClose(ctx context.Context) bool {
	a.StopSend()
	return false
}

// --- Config ---

// LoadConfig returns the current config.
func (a *AppService) LoadConfig() core.AppConfig {
	return a.config
}

// SyncConfig updates the in-memory config without saving to disk.
// Called by the frontend before Send to ensure backend has the latest state.
func (a *AppService) SyncConfig(cfg core.AppConfig) {
	a.config = cfg
}

// SaveConfig persists config to disk and emits a save event with the file path.
func (a *AppService) SaveConfig(cfg core.AppConfig) error {
	a.config = cfg
	err := core.SaveConfig(cfg)
	if err == nil {
		path, _ := core.ConfigPath()
		runtime.EventsEmit(a.ctx, "config:saved", map[string]string{
			"path": path,
		})
	}
	return err
}

// GetConfigPath returns the config file path.
func (a *AppService) GetConfigPath() string {
	path, _ := core.ConfigPath()
	return path
}

// SavePassword stores a password in the OS keychain.
func (a *AppService) SavePassword(authID, password string) error {
	return core.SavePassword(authID, password)
}

// LoadPassword retrieves a password from the OS keychain.
func (a *AppService) LoadPassword(authID string) (string, error) {
	return core.LoadPassword(authID)
}

// HasPassword checks whether a password exists in the OS keychain for the given auth ID.
func (a *AppService) HasPassword(authID string) bool {
	if authID == "" {
		return false
	}
	pw, err := core.LoadPassword(authID)
	return err == nil && pw != ""
}

// --- Load Config From File ---

// LoadConfigFrom opens a file dialog to select a config JSON file and loads it.
func (a *AppService) LoadConfigFrom() (core.AppConfig, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Load Config File",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return a.config, err
	}
	if file == "" {
		return a.config, nil // user cancelled
	}
	cfg, err := core.LoadConfigFrom(file)
	if err != nil {
		return a.config, fmt.Errorf("failed to load config from %s: %w", file, err)
	}
	a.config = cfg
	runtime.EventsEmit(a.ctx, "config:loaded", map[string]string{
		"path": file,
	})
	return cfg, nil
}

// --- Connection Test ---

// TestConnection tests TCP/TLS connectivity to the configured SMTP server.
// Emits smtp:log for live output and test:result with the final outcome.
func (a *AppService) TestConnection() error {
	err := core.TestConnection(a.config.Server, func(direction, line string) {
		runtime.EventsEmit(a.ctx, "smtp:log", map[string]string{
			"direction": direction,
			"line":      line,
		})
	})

	addr := fmt.Sprintf("%s:%d", a.config.Server.SMTP, a.config.Server.Port)
	if err != nil {
		runtime.EventsEmit(a.ctx, "test:result", map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"server":  addr,
		})
	} else {
		runtime.EventsEmit(a.ctx, "test:result", map[string]interface{}{
			"success": true,
			"server":  addr,
		})
	}
	return err
}

// --- Send Control ---

// StartSend begins the mail send process.
func (a *AppService) StartSend() error {
	a.mu.Lock()
	if a.sending {
		a.mu.Unlock()
		return fmt.Errorf("already sending")
	}
	a.sending = true
	a.mu.Unlock()

	// Validate
	if errs := core.ValidateServerConfig(a.config.Server); len(errs) > 0 {
		a.mu.Lock()
		a.sending = false
		a.mu.Unlock()
		return fmt.Errorf("server config error: %s", strings.Join(errs, "; "))
	}
	if errs := core.ValidateMailConfig(a.config.Mail); len(errs) > 0 {
		a.mu.Lock()
		a.sending = false
		a.mu.Unlock()
		return fmt.Errorf("mail config error: %s", strings.Join(errs, "; "))
	}

	// Load password from keychain
	var password string
	if a.config.Server.Auth {
		pw, err := core.LoadPassword(a.config.Server.AuthID)
		if err != nil {
			a.mu.Lock()
			a.sending = false
			a.mu.Unlock()
			return fmt.Errorf("failed to load password: %w", err)
		}
		password = pw
	}

	// Deep copy config and attachments to avoid race with UI changes during send
	serverCfg := a.config.Server
	mailCfg := a.config.Mail
	attachCopy := make([]string, len(a.attachments))
	copy(attachCopy, a.attachments)

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			a.sending = false
			a.cancel = nil
			a.mu.Unlock()
		}()

		var lastSent, lastFailed atomic.Int64

		core.StartSend(ctx, serverCfg, password, mailCfg, attachCopy,
			func(p core.ProgressEvent) {
				lastSent.Store(int64(p.Sent))
				lastFailed.Store(int64(p.Failed))
				runtime.EventsEmit(a.ctx, "progress", p)
			},
			func(r core.SendResult) {
				runtime.EventsEmit(a.ctx, "result", r)
			},
			func(direction, line string) {
				runtime.EventsEmit(a.ctx, "smtp:log", map[string]string{
					"direction": direction,
					"line":      line,
				})
			},
		)

		runtime.EventsEmit(a.ctx, "done", map[string]int{
			"success": int(lastSent.Load()),
			"failed":  int(lastFailed.Load()),
			"total":   mailCfg.MailNumber,
		})
	}()

	return nil
}

// StartSendEML begins the EML file send process.
// mailNumber controls how many times the EML file(s) are sent (cycles through files).
func (a *AppService) StartSendEML(emlFiles []string, from, rcpt string, numberingFrom, numberingTo bool, useHeaderEnvelope, updateMessageID bool, mailNumber, threadNumber, intervalMs int) error {
	a.mu.Lock()
	if a.sending {
		a.mu.Unlock()
		return fmt.Errorf("already sending")
	}
	a.sending = true
	a.mu.Unlock()

	if len(emlFiles) == 0 {
		a.mu.Lock()
		a.sending = false
		a.mu.Unlock()
		return fmt.Errorf("no EML files selected")
	}

	if errs := core.ValidateServerConfig(a.config.Server); len(errs) > 0 {
		a.mu.Lock()
		a.sending = false
		a.mu.Unlock()
		return fmt.Errorf("server config error: %s", strings.Join(errs, "; "))
	}

	var password string
	if a.config.Server.Auth {
		pw, err := core.LoadPassword(a.config.Server.AuthID)
		if err != nil {
			a.mu.Lock()
			a.sending = false
			a.mu.Unlock()
			return fmt.Errorf("failed to load password: %w", err)
		}
		password = pw
	}

	if mailNumber <= 0 {
		mailNumber = len(emlFiles)
	}
	if threadNumber <= 0 {
		threadNumber = 1
	}

	// Deep copy config to avoid race with UI changes during send
	serverCfg := a.config.Server
	customHeaders := make([]core.Header, len(a.config.Mail.CustomHeaders))
	copy(customHeaders, a.config.Mail.CustomHeaders)

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			a.sending = false
			a.cancel = nil
			a.mu.Unlock()
		}()

		var lastSent, lastFailed atomic.Int64

		core.StartSendEML(ctx, serverCfg, password, emlFiles, from, rcpt,
			numberingFrom, numberingTo,
			useHeaderEnvelope, updateMessageID,
			customHeaders,
			mailNumber, threadNumber, intervalMs,
			func(p core.ProgressEvent) {
				lastSent.Store(int64(p.Sent))
				lastFailed.Store(int64(p.Failed))
				runtime.EventsEmit(a.ctx, "progress", p)
			},
			func(r core.SendResult) {
				runtime.EventsEmit(a.ctx, "result", r)
			},
			func(direction, line string) {
				runtime.EventsEmit(a.ctx, "smtp:log", map[string]string{
					"direction": direction,
					"line":      line,
				})
			},
		)

		runtime.EventsEmit(a.ctx, "done", map[string]int{
			"success": int(lastSent.Load()),
			"failed":  int(lastFailed.Load()),
			"total":   mailNumber,
		})
	}()

	return nil
}

// StopSend cancels the current send operation.
func (a *AppService) StopSend() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

// IsSending returns whether a send operation is in progress.
func (a *AppService) IsSending() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sending
}

// --- Window State ---

// SaveWindowState persists the window position and size.
func (a *AppService) SaveWindowState(ws core.WindowState) error {
	a.config.Window = ws
	return core.SaveConfig(a.config)
}

// --- EML Mode ---

// SelectEMLFiles opens a file dialog for selecting .eml files.
func (a *AppService) SelectEMLFiles() ([]string, error) {
	selection, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select EML Files",
		Filters: []runtime.FileFilter{
			{DisplayName: "EML Files (*.eml)", Pattern: "*.eml"},
		},
	})
	if err != nil {
		return nil, err
	}
	return selection, nil
}

// --- EML Preview ---

// ParseEMLPreview parses an .eml file and returns subject, from, to, body for preview.
func (a *AppService) ParseEMLPreview(path string) (core.EMLPreview, error) {
	return core.ParseEMLPreview(path)
}

// --- Attachments (Input mode) ---

// SelectAttachments opens a file dialog for selecting attachment files.
func (a *AppService) SelectAttachments() ([]string, error) {
	selection, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Attachment Files",
		Filters: []runtime.FileFilter{
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(selection) > 0 {
		a.attachments = append(a.attachments, selection...)
	}
	return a.attachments, nil
}

// ClearAttachments removes all stored attachment paths.
func (a *AppService) ClearAttachments() {
	a.attachments = nil
}

// GetAttachments returns the current list of attachment paths.
func (a *AppService) GetAttachments() []string {
	return a.attachments
}

// RemoveAttachment removes an attachment at the given index.
func (a *AppService) RemoveAttachment(index int) []string {
	if index >= 0 && index < len(a.attachments) {
		a.attachments = append(a.attachments[:index], a.attachments[index+1:]...)
	}
	return a.attachments
}

// --- Version ---

// GetVersion returns the current app version string.
func (a *AppService) GetVersion() string {
	return core.AppVersion
}

// CheckVersion checks for updates from the remote server.
func (a *AppService) CheckVersion() core.VersionCheckResult {
	return core.CheckVersion()
}

// --- Log ---

// ClearLog emits a log clear event to the frontend.
func (a *AppService) ClearLog() {
	runtime.EventsEmit(a.ctx, "log:clear", nil)
}
