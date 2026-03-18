package core

// ServerConfig holds SMTP server connection settings.
type ServerConfig struct {
	SMTP       string `json:"smtp"`
	Port       int    `json:"port"`
	TLS        bool   `json:"tls"`
	SSL        bool   `json:"ssl"`
	TLSVersion string `json:"tls_version"` // "1.0", "1.1", "1.2", "1.3" (default: "1.3")
	SkipVerify bool   `json:"skip_verify"` // Skip TLS certificate verification (for self-signed certs)
	Auth       bool   `json:"auth"`
	AuthID     string `json:"auth_id"`
}

// MailConfig holds mail composition and send settings.
type MailConfig struct {
	MailFrom          string   `json:"mail_from"`
	NumberingMailFrom bool     `json:"numbering_mail_from"`
	RcptTo            string   `json:"rcpt_to"`
	NumberingRcptTo   bool     `json:"numbering_rcpt_to"`
	Subject           string   `json:"subject"`
	NumberingSubject  bool     `json:"numbering_subject"`
	TimestampSubject  bool     `json:"timestamp_subject"`
	Body              string   `json:"body"`
	ContentType       string   `json:"content_type"`
	MailNumber        int      `json:"mail_number"`
	ThreadNumber      int      `json:"thread_number"`
	IntervalMs        int      `json:"interval_ms"`
	UseHeaderEnvelope bool     `json:"use_header_envelope"`
	UpdateMessageID   bool     `json:"update_message_id"`
	CustomHeaders     []Header `json:"custom_headers"`
}

// Header is a custom mail header key-value pair.
type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SendMode determines whether mail is composed in UI or loaded from .eml files.
type SendMode string

const (
	SendModeInput SendMode = "input"
	SendModeEML   SendMode = "eml"
)

// WindowState stores the app window position and size.
type WindowState struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// AppConfig is the top-level configuration persisted as JSON.
type AppConfig struct {
	Server ServerConfig `json:"server"`
	Mail   MailConfig   `json:"mail"`
	Window WindowState  `json:"window"`
	Theme  string       `json:"theme"` // "dark" or "light"
}

// SendResult is the outcome of sending a single mail.
type SendResult struct {
	Index   int    `json:"index"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Subject string `json:"subject,omitempty"`
}

// ProgressEvent reports real-time send progress.
type ProgressEvent struct {
	Sent   int `json:"sent"`
	Failed int `json:"failed"`
	Total  int `json:"total"`
}

// EMLPreview holds parsed EML data for preview display.
type EMLPreview struct {
	Subject     string `json:"subject"`
	From        string `json:"from"`
	To          string `json:"to"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() AppConfig {
	return AppConfig{
		Server: ServerConfig{
			SMTP:       "localhost",
			Port:       25,
			TLSVersion: "1.3",
			SkipVerify: true,
		},
		Mail: MailConfig{
			ContentType:  "text/plain",
			MailNumber:   1,
			ThreadNumber: 1,
			IntervalMs:   0,
		},
		Window: WindowState{
			Width:  1024,
			Height: 720,
		},
		Theme: "dark",
	}
}
