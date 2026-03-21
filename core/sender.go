package core

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	netmail "net/mail"
	"net/smtp"
	"os"
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
// Both min and max are set to the same value so the exact selected version is used.
func parseTLSVersion(ver string) (min, max uint16) {
	switch ver {
	case "1.0":
		return tls.VersionTLS10, tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11, tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12, tls.VersionTLS12
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

// authMask matches AUTH PLAIN/LOGIN credentials in SMTP log lines and masks them.
var authMask = regexp.MustCompile(`(?i)(AUTH\s+(?:PLAIN|LOGIN)\s+).+`)

func (l *smtpLogger) log(dir string, entry gomaillog.Log) {
	if l.onLog == nil {
		return
	}
	msg := fmt.Sprintf(entry.Format, entry.Messages...)
	// go-mail may produce multi-line output; split and send each line
	for _, line := range strings.Split(strings.TrimRight(msg, "\r\n"), "\n") {
		line = strings.TrimRight(line, "\r")
		if line != "" {
			// Mask AUTH credentials (Base64-encoded ID/PW) in SMTP log
			line = authMask.ReplaceAllString(line, "${1}****")
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

// reservedHeaders is a set of standard headers that should not be overwritten by custom headers.
var reservedHeaders = map[string]bool{
	"from": true, "to": true, "cc": true, "bcc": true,
	"subject": true, "date": true, "message-id": true,
	"reply-to": true, "return-path": true, "sender": true,
}

// sanitizeHeader removes \r and \n from header key/value to prevent CRLF injection.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

// applyCustomHeaders applies custom headers to a message with CRLF sanitization.
// Reserved headers (From, To, Subject, etc.) are skipped with a warning log.
func applyCustomHeaders(m *gomail.Msg, headers []Header, onLog func(direction, line string)) {
	for _, h := range headers {
		if h.Key == "" {
			continue
		}
		key := sanitizeHeader(h.Key)
		value := sanitizeHeader(h.Value)

		if reservedHeaders[strings.ToLower(key)] {
			if onLog != nil {
				onLog("warn", fmt.Sprintf("Custom header %q is a reserved header — skipped (use envelope fields instead)", key))
			}
			continue
		}

		m.SetGenHeader(gomail.Header(key), value)
	}
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

// SendEMLRaw sends an .eml file via SMTP by transmitting the raw EML bytes after DATA.
// Unlike the old SendEML (which parsed and reassembled via go-mail), this preserves the
// original EML content exactly — MIME structure, attachments, boundaries, and nested EMLs.
// Only Message-ID and custom headers are modified at the byte level in the header section.
func SendEMLRaw(server ServerConfig, password string, from, rcpt, emlPath string, useHeaderEnvelope bool, updateMessageID bool, customHeaders []Header, onLog func(direction, line string)) (usedFrom, usedTo string, err error) {
	log := func(dir, msg string) {
		if onLog != nil {
			onLog(dir, msg)
		}
	}

	// 1. Read EML file as raw bytes
	log("info", fmt.Sprintf("Reading EML file: %s", emlPath))
	rawBytes, err := os.ReadFile(emlPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read EML file: %w", err)
	}

	// 2. Split header and body at first blank line (\r\n\r\n or \n\n)
	headerPart, bodyPart := splitEMLHeaderBody(rawBytes)

	// 3. Determine envelope addresses
	if useHeaderEnvelope {
		emlFrom := extractHeaderValue(headerPart, "From")
		emlTo := extractHeaderValue(headerPart, "To")
		if emlFrom == "" {
			return "", "", fmt.Errorf("EML file has no From header — cannot use header envelope")
		}
		if emlTo == "" {
			return "", "", fmt.Errorf("EML file has no To header — cannot use header envelope")
		}
		usedFrom = extractAddr(emlFrom)
		usedTo = extractAddr(emlTo)
		log("info", fmt.Sprintf("Using header envelope: From=%q, To=%q", usedFrom, usedTo))
	} else {
		if from == "" {
			return "", "", fmt.Errorf("From address is required when Use Header Envelope is off")
		}
		if rcpt == "" {
			return "", "", fmt.Errorf("To address is required when Use Header Envelope is off")
		}
		usedFrom = from
		usedTo = rcpt
		log("info", fmt.Sprintf("Envelope override: From=%q, To=%q", usedFrom, usedTo))
	}

	// 4. Modify header section (Message-ID, Custom Headers)
	if updateMessageID {
		newID := generateMessageID()
		headerPart = replaceOrInsertHeader(headerPart, "Message-ID", "<"+newID+">")
		log("info", fmt.Sprintf("Message-ID updated: <%s>", newID))
	}
	for _, h := range customHeaders {
		if h.Key == "" {
			continue
		}
		key := sanitizeHeader(h.Key)
		value := sanitizeHeader(h.Value)
		if reservedHeaders[strings.ToLower(key)] {
			log("warn", fmt.Sprintf("Custom header %q is reserved — skipped", key))
			continue
		}
		headerPart = append(headerPart, []byte(key+": "+value+"\r\n")...)
	}

	// 5. Reassemble modified EML
	emlData := append(headerPart, bodyPart...)

	// 6. SMTP connection and raw send
	addr := fmt.Sprintf("%s:%d", server.SMTP, server.Port)
	tlsCfg := &tls.Config{
		ServerName:         server.SMTP,
		InsecureSkipVerify: server.SkipVerify,
	}
	tlsCfg.MinVersion, tlsCfg.MaxVersion = parseTLSVersion(server.TLSVersion)

	var smtpClient *smtp.Client

	if server.SSL {
		// Implicit SSL: TLS first, then SMTP
		log("client", fmt.Sprintf("Connecting to %s (SSL/TLS)...", addr))
		tlsConn, err := tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp", addr, tlsCfg)
		if err != nil {
			log("error", fmt.Sprintf("SSL connection failed: %v", err))
			return "", "", fmt.Errorf("SSL connection failed: %w", err)
		}
		defer tlsConn.Close()
		smtpClient, err = smtp.NewClient(tlsConn, server.SMTP)
		if err != nil {
			log("error", fmt.Sprintf("SMTP client creation failed: %v", err))
			return "", "", fmt.Errorf("SMTP client creation failed: %w", err)
		}
	} else {
		// Plain or STARTTLS
		log("client", fmt.Sprintf("Connecting to %s...", addr))
		conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
		if err != nil {
			log("error", fmt.Sprintf("TCP connection failed: %v", err))
			return "", "", fmt.Errorf("TCP connection failed: %w", err)
		}
		defer conn.Close()
		smtpClient, err = smtp.NewClient(conn, server.SMTP)
		if err != nil {
			log("error", fmt.Sprintf("SMTP client creation failed: %v", err))
			return "", "", fmt.Errorf("SMTP client creation failed: %w", err)
		}
	}
	defer smtpClient.Quit()

	// EHLO
	log("client", fmt.Sprintf("EHLO %s", server.SMTP))
	if err := smtpClient.Hello(server.SMTP); err != nil {
		log("error", fmt.Sprintf("EHLO failed: %v", err))
		return "", "", fmt.Errorf("EHLO failed: %w", err)
	}

	// STARTTLS
	if server.TLS && !server.SSL {
		log("client", "STARTTLS")
		if err := smtpClient.StartTLS(tlsCfg); err != nil {
			log("error", fmt.Sprintf("STARTTLS failed: %v", err))
			return "", "", fmt.Errorf("STARTTLS failed: %w", err)
		}
		log("info", "TLS connection established")
	}

	// AUTH
	if server.Auth && server.AuthID != "" && password != "" {
		log("client", fmt.Sprintf("AUTH PLAIN %s ****", server.AuthID))
		auth := smtp.PlainAuth("", server.AuthID, password, server.SMTP)
		if err := smtpClient.Auth(auth); err != nil {
			log("error", fmt.Sprintf("AUTH failed: %v", err))
			return "", "", fmt.Errorf("AUTH failed: %w", err)
		}
		log("info", "Authentication successful")
	}

	// MAIL FROM
	log("client", fmt.Sprintf("MAIL FROM:<%s>", usedFrom))
	if err := smtpClient.Mail(usedFrom); err != nil {
		log("error", fmt.Sprintf("MAIL FROM failed: %v", err))
		return "", "", fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// RCPT TO
	log("client", fmt.Sprintf("RCPT TO:<%s>", usedTo))
	if err := smtpClient.Rcpt(usedTo); err != nil {
		log("error", fmt.Sprintf("RCPT TO failed: %v", err))
		return "", "", fmt.Errorf("RCPT TO failed: %w", err)
	}

	// DATA
	log("client", "DATA")
	writer, err := smtpClient.Data()
	if err != nil {
		log("error", fmt.Sprintf("DATA failed: %v", err))
		return "", "", fmt.Errorf("DATA command failed: %w", err)
	}
	if _, err := writer.Write(emlData); err != nil {
		log("error", fmt.Sprintf("DATA write failed: %v", err))
		return "", "", fmt.Errorf("DATA write failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		log("error", fmt.Sprintf("DATA close failed: %v", err))
		return "", "", fmt.Errorf("send failed: %w", err)
	}
	log("info", "Message sent successfully")

	return usedFrom, usedTo, nil
}

// splitEMLHeaderBody splits raw EML bytes into header and body parts at the first blank line.
func splitEMLHeaderBody(raw []byte) (header, body []byte) {
	// Try \r\n\r\n first (standard), then \n\n
	if idx := bytes.Index(raw, []byte("\r\n\r\n")); idx >= 0 {
		return raw[:idx+2], raw[idx+2:] // header includes trailing \r\n, body starts with \r\n
	}
	if idx := bytes.Index(raw, []byte("\n\n")); idx >= 0 {
		return raw[:idx+1], raw[idx+1:]
	}
	// No body found — entire file is headers
	return raw, nil
}

// extractHeaderValue extracts the value of a header field from the header section bytes.
// Handles folded headers (continuation lines starting with space/tab).
func extractHeaderValue(header []byte, name string) string {
	prefix := strings.ToLower(name) + ":"
	lines := bytes.Split(header, []byte("\n"))
	var result string
	found := false
	for _, line := range lines {
		lineStr := strings.TrimRight(string(line), "\r")
		if found {
			// Check for folded continuation (starts with space or tab)
			if len(lineStr) > 0 && (lineStr[0] == ' ' || lineStr[0] == '\t') {
				result += " " + strings.TrimSpace(lineStr)
				continue
			}
			break
		}
		if strings.HasPrefix(strings.ToLower(lineStr), prefix) {
			result = strings.TrimSpace(lineStr[len(prefix):])
			found = true
		}
	}
	return result
}

// replaceOrInsertHeader replaces a header value in the header section, or inserts it if not found.
// Only operates on the header part (before the first blank line) to avoid modifying nested EML attachments.
func replaceOrInsertHeader(header []byte, name, value string) []byte {
	prefix := strings.ToLower(name) + ":"
	lines := bytes.Split(header, []byte("\n"))
	var result [][]byte
	replaced := false
	skip := false
	for _, line := range lines {
		lineStr := strings.TrimRight(string(line), "\r")
		if skip {
			// Skip folded continuation lines of the replaced header
			if len(lineStr) > 0 && (lineStr[0] == ' ' || lineStr[0] == '\t') {
				continue
			}
			skip = false
		}
		if !replaced && strings.HasPrefix(strings.ToLower(lineStr), prefix) {
			result = append(result, []byte(name+": "+value+"\r"))
			replaced = true
			skip = true // skip any continuation lines
			continue
		}
		result = append(result, line)
	}
	if !replaced {
		// Insert before the last line (which is empty, marking end of headers)
		newHeader := []byte(name + ": " + value + "\r\n")
		header = append(header, newHeader...)
		return header
	}
	return bytes.Join(result, []byte("\n"))
}

// generateMessageID creates a new unique Message-ID.
func generateMessageID() string {
	timestamp := time.Now().UnixNano()
	// Use base64 of timestamp for a compact unique ID
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = byte(timestamp & 0xff)
		timestamp >>= 8
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	return fmt.Sprintf("%s.%d@dmailsender", encoded, time.Now().UnixMicro()%100000)
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
	m := gomail.NewMsg(gomail.WithNoDefaultUserAgent())

	// Apply numbering
	from := mail.MailFrom
	if mail.NumberingMailFrom {
		from = applyNumbering(from, index)
	}

	rcpt := mail.RcptTo
	if mail.NumberingRcptTo {
		rcpt = applyNumbering(rcpt, index)
	}

	// From/To are always required in Input mode
	if from == "" {
		return fmt.Errorf("From address is required")
	}
	if rcpt == "" {
		return fmt.Errorf("To address is required")
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

	// Custom headers (sanitized, reserved headers skipped with warning)
	applyCustomHeaders(m, mail.CustomHeaders, onLog)

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
		if onLog != nil {
			onLog("error", fmt.Sprintf("Send failed: %v", err))
		}
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
