package core

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	netmail "net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	gomail "github.com/wneessen/go-mail"
	gomaillog "github.com/wneessen/go-mail/log"
)

// TestConnection tests TCP connectivity to the SMTP server and optionally TLS handshake.
// It returns a log of what happened and any error encountered.
func TestConnection(server ServerConfig, onLog func(direction, line string)) error {
	addr := fmt.Sprintf("%s:%d", server.SMTP, server.Port)

	onLog("info", fmt.Sprintf("Connecting to %s...", addr))

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		onLog("error", fmt.Sprintf("TCP connection failed: %v", err))
		return fmt.Errorf("TCP connection failed: %w", err)
	}
	defer conn.Close()

	onLog("info", fmt.Sprintf("TCP connection established to %s", addr))

	tlsCfg := &tls.Config{
		ServerName:         server.SMTP,
		InsecureSkipVerify: server.SkipVerify,
	}
	tlsCfg.MinVersion, tlsCfg.MaxVersion = parseTLSVersion(server.TLSVersion)

	buf := make([]byte, 1024)

	if server.SSL {
		// Implicit SSL (port 465): TLS handshake FIRST, then read greeting
		onLog("info", "Starting TLS handshake (Implicit SSL)...")
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			onLog("error", fmt.Sprintf("TLS handshake failed: %v", err))
			return fmt.Errorf("TLS handshake failed: %w", err)
		}
		state := tlsConn.ConnectionState()
		onLog("info", fmt.Sprintf("TLS handshake OK — version: %s, cipher: %s",
			tlsVersionName(state.Version), tls.CipherSuiteName(state.CipherSuite)))

		// Now read greeting over the TLS connection
		tlsConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := tlsConn.Read(buf)
		if err != nil {
			onLog("error", fmt.Sprintf("Failed to read greeting: %v", err))
			return fmt.Errorf("failed to read greeting: %w", err)
		}
		onLog("server", strings.TrimSpace(string(buf[:n])))
	} else {
		// Plain or STARTTLS: read greeting first
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			onLog("error", fmt.Sprintf("Failed to read greeting: %v", err))
			return fmt.Errorf("failed to read greeting: %w", err)
		}
		onLog("server", strings.TrimSpace(string(buf[:n])))

		if server.TLS {
			// STARTTLS: send EHLO then STARTTLS command
			onLog("client", "EHLO test")
			fmt.Fprintf(conn, "EHLO test\r\n")
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, _ = conn.Read(buf)
			onLog("server", strings.TrimSpace(string(buf[:n])))

			onLog("client", "STARTTLS")
			fmt.Fprintf(conn, "STARTTLS\r\n")
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, _ = conn.Read(buf)
			resp := strings.TrimSpace(string(buf[:n]))
			onLog("server", resp)

			if !strings.HasPrefix(resp, "220") {
				onLog("error", "Server did not accept STARTTLS")
				return fmt.Errorf("STARTTLS rejected: %s", resp)
			}

			tlsConn := tls.Client(conn, tlsCfg)
			if err := tlsConn.Handshake(); err != nil {
				onLog("error", fmt.Sprintf("TLS handshake failed: %v", err))
				return fmt.Errorf("TLS handshake failed: %w", err)
			}
			state := tlsConn.ConnectionState()
			onLog("info", fmt.Sprintf("TLS handshake OK — version: %s, cipher: %s",
				tlsVersionName(state.Version), tls.CipherSuiteName(state.CipherSuite)))
		}
	}

	onLog("info", "Connection test completed successfully.")
	return nil
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}

// digitSuffix matches trailing digits before @ in an email address.
var digitSuffix = regexp.MustCompile(`(\d+)(@.+)$`)

// applyNumbering replaces trailing digits before @ with the given index,
// preserving the original digit width with zero-padding.
// If no digits exist, the index is inserted before @.
func applyNumbering(addr string, index int) string {
	if digitSuffix.MatchString(addr) {
		return digitSuffix.ReplaceAllStringFunc(addr, func(match string) string {
			parts := digitSuffix.FindStringSubmatch(match)
			width := len(parts[1])
			return fmt.Sprintf("%0*d%s", width, index, parts[2])
		})
	}

	// No digits found: insert index before @
	atIdx := strings.LastIndex(addr, "@")
	if atIdx == -1 {
		return addr + strconv.Itoa(index)
	}
	return addr[:atIdx] + strconv.Itoa(index) + addr[atIdx:]
}

// applyNumberingSubject replaces trailing digits in the subject with the index.
var digitEnd = regexp.MustCompile(`(\d+)$`)

func applyNumberingSubject(subject string, index int) string {
	if digitEnd.MatchString(subject) {
		return digitEnd.ReplaceAllStringFunc(subject, func(match string) string {
			width := len(match)
			return fmt.Sprintf("%0*d", width, index)
		})
	}
	return subject + strconv.Itoa(index)
}

// parseTLSVersion returns min and max TLS version from a string like "1.0", "1.2", "1.3".
// Default is TLS 1.3 only. When a lower version is selected, it sets that as min and 1.3 as max.
func parseTLSVersion(ver string) (min, max uint16) {
	switch ver {
	case "1.0":
		return tls.VersionTLS10, tls.VersionTLS13
	case "1.1":
		return tls.VersionTLS11, tls.VersionTLS13
	case "1.2":
		return tls.VersionTLS12, tls.VersionTLS13
	case "1.3":
		return tls.VersionTLS13, tls.VersionTLS13
	default:
		return tls.VersionTLS13, tls.VersionTLS13
	}
}

// smtpLogger implements go-mail/log.Logger and forwards log lines to onLog callback.
type smtpLogger struct {
	onLog func(direction, line string)
}

func (l *smtpLogger) log(dir string, entry gomaillog.Log) {
	if l.onLog == nil {
		return
	}
	msg := fmt.Sprintf(entry.Format, entry.Messages...)
	// go-mail may produce multi-line output; split and send each line
	for _, line := range strings.Split(strings.TrimRight(msg, "\r\n"), "\n") {
		line = strings.TrimRight(line, "\r")
		if line != "" {
			l.onLog(dir, line)
		}
	}
}

func (l *smtpLogger) Debugf(entry gomaillog.Log) {
	dir := "client"
	if entry.Direction == gomaillog.DirServerToClient {
		dir = "server"
	}
	l.log(dir, entry)
}

func (l *smtpLogger) Infof(entry gomaillog.Log) {
	l.log("info", entry)
}

func (l *smtpLogger) Warnf(entry gomaillog.Log) {
	l.log("warn", entry)
}

func (l *smtpLogger) Errorf(entry gomaillog.Log) {
	l.log("error", entry)
}

// extractAddr extracts the bare email address from a header value like "Display Name <addr@example.com>".
// If parsing fails, returns the original string as-is.
func extractAddr(raw string) string {
	addr, err := netmail.ParseAddress(raw)
	if err != nil {
		// Fallback: return trimmed original
		return strings.TrimSpace(raw)
	}
	return addr.Address
}

// SendEML sends an .eml file via SMTP using EMLToMsgFromFile.
// Returns the actual from/to addresses used for sending (for accurate logging).
// When useHeaderEnvelope=true, the EML's own From/To headers are used as SMTP envelope addresses.
// When useHeaderEnvelope=false, the from/rcpt parameters override the SMTP envelope addresses.
func SendEML(server ServerConfig, password string, from, rcpt, emlPath string, useHeaderEnvelope bool, updateMessageID bool, customHeaders []Header, onLog func(direction, line string)) (usedFrom, usedTo string, err error) {
	if onLog != nil {
		onLog("info", fmt.Sprintf("Parsing EML file: %s", emlPath))
	}

	m, err := gomail.EMLToMsgFromFile(emlPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse EML file: %w", err)
	}

	if useHeaderEnvelope {
		// Use EML header From/To as SMTP envelope addresses (no UI override)
		// - From header → extract bare address → set as EnvelopeFrom (MAIL FROM)
		// - To header → keep as-is (go-mail uses it for RCPT TO automatically)
		emlFrom := ""
		emlTo := ""
		if addrs := m.GetFromString(); len(addrs) > 0 {
			emlFrom = extractAddr(addrs[0])
		}
		if addrs := m.GetToString(); len(addrs) > 0 {
			emlTo = extractAddr(addrs[0])
		}

		if onLog != nil {
			onLog("info", fmt.Sprintf("EML parsed OK — Using header envelope: From=%q, To=%q", emlFrom, emlTo))
		}

		// Set envelope sender from EML header (bare address for SMTP MAIL FROM)
		if emlFrom != "" {
			if err := m.EnvelopeFrom(emlFrom); err != nil {
				return "", "", fmt.Errorf("invalid EML header From for envelope: %w", err)
			}
		}
		// To header is not overridden — go-mail uses the EML's original To for RCPT TO

		usedFrom = emlFrom
		usedTo = emlTo
	} else {
		// Override with user-provided values
		if onLog != nil {
			onLog("info", fmt.Sprintf("EML parsed OK — From override: %q, To override: %q", from, rcpt))
		}

		if from != "" {
			if err := m.From(from); err != nil {
				return "", "", fmt.Errorf("invalid from: %w", err)
			}
		}
		if rcpt != "" {
			if err := m.To(rcpt); err != nil {
				return "", "", fmt.Errorf("invalid to: %w", err)
			}
		}

		usedFrom = from
		usedTo = rcpt
	}

	if updateMessageID {
		m.SetMessageID()
	}

	// Custom headers (appended to existing EML headers)
	for _, h := range customHeaders {
		if h.Key != "" {
			m.SetGenHeader(gomail.Header(h.Key), h.Value)
		}
	}

	// Build client options
	opts := buildClientOpts(server, password, onLog)

	client, err := gomail.NewClient(server.SMTP, opts...)
	if err != nil {
		return "", "", fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if onLog != nil {
		client.SetLogger(&smtpLogger{onLog: onLog})
	}

	if err := client.DialAndSend(m); err != nil {
		return "", "", fmt.Errorf("send failed: %w", err)
	}

	return usedFrom, usedTo, nil
}

// buildClientOpts creates common go-mail client options from server config.
func buildClientOpts(server ServerConfig, password string, onLog func(direction, line string)) []gomail.Option {
	var opts []gomail.Option
	opts = append(opts, gomail.WithPort(server.Port))

	tlsCfg := &tls.Config{
		ServerName:         server.SMTP,
		InsecureSkipVerify: server.SkipVerify,
	}
	tlsCfg.MinVersion, tlsCfg.MaxVersion = parseTLSVersion(server.TLSVersion)

	if server.SSL {
		opts = append(opts, gomail.WithSSLPort(false))
		opts = append(opts, gomail.WithTLSConfig(tlsCfg))
	} else if server.TLS {
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.TLSMandatory))
		opts = append(opts, gomail.WithTLSConfig(tlsCfg))
	} else {
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.NoTLS))
	}

	if server.Auth && server.AuthID != "" && password != "" {
		opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthPlain))
		opts = append(opts, gomail.WithUsername(server.AuthID))
		opts = append(opts, gomail.WithPassword(password))
	}

	if onLog != nil {
		opts = append(opts, gomail.WithDebugLog())
	}

	return opts
}

// SendOne sends a single mail message according to the given config and index.
func SendOne(server ServerConfig, password string, mail MailConfig, index int, attachments []string, onLog func(direction, line string)) error {
	m := gomail.NewMsg()

	// Apply numbering
	from := mail.MailFrom
	if mail.NumberingMailFrom {
		from = applyNumbering(from, index)
	}

	rcpt := mail.RcptTo
	if mail.NumberingRcptTo {
		rcpt = applyNumbering(rcpt, index)
	}

	subject := mail.Subject
	if mail.NumberingSubject {
		subject = applyNumberingSubject(subject, index)
	}
	if mail.TimestampSubject {
		subject = subject + " (" + time.Now().Format("2006-01-02 15:04:05") + ")"
	}

	if err := m.From(from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if err := m.To(rcpt); err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	m.Subject(subject)

	// Custom headers
	for _, h := range mail.CustomHeaders {
		if h.Key != "" {
			m.SetGenHeader(gomail.Header(h.Key), h.Value)
		}
	}

	// Body
	if mail.ContentType == "text/html" {
		m.SetBodyString(gomail.TypeTextHTML, mail.Body)
	} else {
		m.SetBodyString(gomail.TypeTextPlain, mail.Body)
	}

	// Envelope control
	if mail.UseHeaderEnvelope {
		if err := m.EnvelopeFrom(from); err != nil {
			return fmt.Errorf("invalid envelope from: %w", err)
		}
	}
	if mail.UpdateMessageID {
		m.SetMessageID()
	}

	// Attach files
	for _, path := range attachments {
		m.AttachFile(path)
	}

	// Build client options
	opts := buildClientOpts(server, password, onLog)

	client, err := gomail.NewClient(server.SMTP, opts...)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	// Attach custom logger that forwards to onLog callback
	if onLog != nil {
		client.SetLogger(&smtpLogger{onLog: onLog})
	}

	if err := client.DialAndSend(m); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}

	return nil
}

// ParseEMLPreview parses an .eml file and returns a preview (subject, from, to, body).
func ParseEMLPreview(emlPath string) (EMLPreview, error) {
	preview := EMLPreview{}
	m, err := gomail.EMLToMsgFromFile(emlPath)
	if err != nil {
		return preview, fmt.Errorf("failed to parse EML: %w", err)
	}

	// MIME word decoder for encoded headers (e.g., =?UTF-8?B?...?=)
	dec := new(mime.WordDecoder)

	// Extract headers
	if addrs := m.GetFromString(); len(addrs) > 0 {
		preview.From = decodeMIME(dec, strings.Join(addrs, ", "))
	}
	if addrs := m.GetToString(); len(addrs) > 0 {
		preview.To = decodeMIME(dec, strings.Join(addrs, ", "))
	}
	if gens := m.GetGenHeader(gomail.HeaderSubject); len(gens) > 0 {
		preview.Subject = decodeMIME(dec, gens[0])
	}

	// Extract body from first part
	parts := m.GetParts()
	for _, p := range parts {
		preview.ContentType = string(p.GetContentType())
		content, err := p.GetContent()
		if err == nil {
			preview.Body = string(content)
		}
		break
	}

	return preview, nil
}

// decodeMIME decodes MIME-encoded header values (e.g., =?UTF-8?B?...?=).
// If decoding fails, returns the original string.
func decodeMIME(dec *mime.WordDecoder, s string) string {
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}
