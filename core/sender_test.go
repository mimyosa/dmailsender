package core

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// applyNumbering (email address)
// ---------------------------------------------------------------------------

func TestApplyNumbering(t *testing.T) {
	// applyNumbering always appends "-{index}" to the local part (before @).
	// It never modifies existing characters — domain digits, local-part digits,
	// and IP domains are all preserved.
	tests := []struct {
		name  string
		input string
		index int
		want  string
	}{
		// Basic cases
		{"plain local, index 3", "user@example.com", 3, "user-3@example.com"},
		{"plain local, index 0", "user@example.com", 0, "user-0@example.com"},

		// Local part already contains digits (must be preserved)
		{"local ends with digits", "user001@example.com", 5, "user001-5@example.com"},
		{"local ends with single digit", "aimaya2@jiran.com", 0, "aimaya2-0@jiran.com"},
		{"local ends with single digit, idx1", "aimaya2@jiran.com", 1, "aimaya2-1@jiran.com"},
		{"local has mixed digits", "a1b2c3@x.com", 9, "a1b2c3-9@x.com"},
		{"local starts with digit", "2user@example.com", 5, "2user-5@example.com"},
		{"local middle digit only", "u1ser@example.com", 5, "u1ser-5@example.com"},

		// Domain contains digits (must NOT be touched)
		{"domain with trailing digit", "user@jiran2.com", 5, "user-5@jiran2.com"},
		{"domain starts with digit", "user@2example.com", 5, "user-5@2example.com"},
		{"IPv4 literal domain", "user@192.168.1.1", 7, "user-7@192.168.1.1"},
		{"IPv4 bracketed domain", "user@[10.0.0.1]", 3, "user-3@[10.0.0.1]"},
		{"subdomain with digits", "user@mx1.corp.jiran2.com", 4, "user-4@mx1.corp.jiran2.com"},

		// Missing @ (fallback path)
		{"no at sign", "no-at-sign", 3, "no-at-sign-3"},
		{"empty string", "", 42, "-42"},

		// Edge indices
		{"large index", "user@example.com", 999999, "user-999999@example.com"},
		{"negative index", "user@example.com", -1, "user--1@example.com"},

		// Special characters in local part
		{"local with dot", "first.last@example.com", 5, "first.last-5@example.com"},
		{"local with plus tag", "user+tag@example.com", 5, "user+tag-5@example.com"},
		{"local with underscore", "user_name@example.com", 5, "user_name-5@example.com"},
		{"local with hyphen", "user-name@example.com", 5, "user-name-5@example.com"},

		// Multiple @ signs — uses LastIndex so the final @ wins
		{"multiple at signs", "weird@name@example.com", 3, "weird@name-3@example.com"},

		// Unicode (Go strings are bytes; @ is ASCII so this should still work)
		{"unicode local", "사용자@example.com", 7, "사용자-7@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyNumbering(tt.input, tt.index)
			if got != tt.want {
				t.Errorf("applyNumbering(%q, %d) = %q, want %q", tt.input, tt.index, got, tt.want)
			}

			// Invariant: the original input must appear as a substring (modulo the inserted "-N")
			// i.e. no existing characters are altered.
			withoutSuffix := strings.Replace(got, fmt.Sprintf("-%d", tt.index), "", 1)
			if withoutSuffix != tt.input {
				t.Errorf("invariant broken: removing -%d from %q -> %q, expected %q",
					tt.index, got, withoutSuffix, tt.input)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// applyNumberingSubject (subject line)
// ---------------------------------------------------------------------------

func TestApplyNumberingSubject(t *testing.T) {
	// applyNumberingSubject always appends "-{index}" to the subject as-is.
	// Trailing digits (years, versions) are never touched.
	tests := []struct {
		name  string
		input string
		index int
		want  string
	}{
		{"plain text", "Test Message", 3, "Test Message-3"},
		{"already has trailing digits", "Test Message 001", 5, "Test Message 001-5"},
		{"year preserved", "Report 2024", 5, "Report 2024-5"},
		{"version preserved", "Version 1.2", 7, "Version 1.2-7"},
		{"empty subject", "", 0, "-0"},
		{"whitespace only", "   ", 1, "   -1"},
		{"trailing space", "Subject ", 2, "Subject -2"},
		{"korean", "테스트 메일", 9, "테스트 메일-9"},
		{"korean with digit", "테스트 100번째", 5, "테스트 100번째-5"},
		{"special chars", "[URGENT] Re: test?", 3, "[URGENT] Re: test?-3"},
		{"emoji", "🚀 Launch", 1, "🚀 Launch-1"},
		{"already ends with dash-digit", "Task-1", 5, "Task-1-5"},
		{"large index", "Msg", 999999, "Msg-999999"},
		{"negative index", "Msg", -1, "Msg--1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyNumberingSubject(tt.input, tt.index)
			if got != tt.want {
				t.Errorf("applyNumberingSubject(%q, %d) = %q, want %q", tt.input, tt.index, got, tt.want)
			}
			// Invariant: result must start with original input
			if !strings.HasPrefix(got, tt.input) {
				t.Errorf("invariant broken: %q is not a prefix of %q", tt.input, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Subject composition: NUM + Timestamp ordering
// ---------------------------------------------------------------------------

// buildSubject mirrors the exact composition logic used in SendOne:
//
//	if NumberingSubject  -> apply numbering
//	if TimestampSubject  -> append " (YYYY-MM-DD HH:MM:SS)"
//
// The expected final form is: "{subject}-{num} ({timestamp})"
func buildSubject(base string, index int, useNum, useTs bool) string {
	s := base
	if useNum {
		s = applyNumberingSubject(s, index)
	}
	if useTs {
		s = s + " (" + time.Now().Format("2006-01-02 15:04:05") + ")"
	}
	return s
}

func TestSubjectNumPlusTimestamp(t *testing.T) {
	// Regex for the timestamp suffix portion.
	tsRe := regexp.MustCompile(` \(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\)$`)

	tests := []struct {
		name      string
		base      string
		index     int
		useNum    bool
		useTs     bool
		wantExact string // when useTs == false, we can check exact
	}{
		{"num only", "Hello", 5, true, false, "Hello-5"},
		{"num only, year preserved", "Report 2024", 3, true, false, "Report 2024-3"},
		{"timestamp only", "Hello", 0, false, true, ""},
		{"both on", "Hello", 7, true, true, ""},
		{"both on with year", "Report 2024", 2, true, true, ""},
		{"neither", "Hello", 0, false, false, "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSubject(tt.base, tt.index, tt.useNum, tt.useTs)

			// Build the expected prefix (without timestamp)
			wantPrefix := tt.base
			if tt.useNum {
				wantPrefix = tt.base + "-" + fmt.Sprintf("%d", tt.index)
			}

			if tt.useTs {
				// Prefix check
				if !strings.HasPrefix(got, wantPrefix+" (") {
					t.Errorf("expected prefix %q, got %q", wantPrefix+" (", got)
				}
				// Suffix check (format)
				if !tsRe.MatchString(got) {
					t.Errorf("timestamp suffix missing or malformed in %q", got)
				}
				// Ordering check: NUM must appear before timestamp
				if tt.useNum {
					numIdx := strings.Index(got, "-"+fmt.Sprintf("%d", tt.index))
					tsIdx := strings.Index(got, " (")
					if numIdx == -1 || tsIdx == -1 || numIdx > tsIdx {
						t.Errorf("NUM must come before timestamp in %q", got)
					}
				}
			} else {
				if got != tt.wantExact {
					t.Errorf("got %q, want %q", got, tt.wantExact)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Broadcast recipients + per-address numbering
// ---------------------------------------------------------------------------

// applyNumberingToList mirrors the per-recipient numbering logic in SendOne:
//
//	rcpts := parseRecipients(mail.RcptTo)
//	if mail.NumberingRcptTo {
//	    for i, r := range rcpts { rcpts[i] = applyNumbering(r, index) }
//	}
func applyNumberingToList(rcptTo string, index int, useNum bool) []string {
	rcpts := parseRecipients(rcptTo)
	if useNum {
		for i, r := range rcpts {
			rcpts[i] = applyNumbering(r, index)
		}
	}
	return rcpts
}

func TestBroadcastRecipientsWithNumbering(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		index  int
		useNum bool
		want   []string
	}{
		{
			"single, no num",
			"user@example.com",
			5, false,
			[]string{"user@example.com"},
		},
		{
			"single, with num",
			"user@example.com",
			5, true,
			[]string{"user-5@example.com"},
		},
		{
			"broadcast no num",
			"a@x.com, b@y.com, c@z.com",
			5, false,
			[]string{"a@x.com", "b@y.com", "c@z.com"},
		},
		{
			"broadcast with num — the reported bug case",
			"aimaya@jiran.com, aimaya2@jiran.com, aimaya@jiran2.com",
			0, true,
			[]string{"aimaya-0@jiran.com", "aimaya2-0@jiran.com", "aimaya-0@jiran2.com"},
		},
		{
			"broadcast with num — index 1",
			"aimaya@jiran.com, aimaya2@jiran.com, aimaya@jiran2.com",
			1, true,
			[]string{"aimaya-1@jiran.com", "aimaya2-1@jiran.com", "aimaya-1@jiran2.com"},
		},
		{
			"semicolon separator",
			"a@x.com; b@y.com",
			3, true,
			[]string{"a-3@x.com", "b-3@y.com"},
		},
		{
			"mixed separators",
			"a@x.com, b@y.com; c@z.com",
			2, true,
			[]string{"a-2@x.com", "b-2@y.com", "c-2@z.com"},
		},
		{
			"extra whitespace and empty slots",
			" a@x.com ,  , b@y.com , ",
			4, true,
			[]string{"a-4@x.com", "b-4@y.com"},
		},
		{
			"empty input",
			"",
			0, false,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyNumberingToList(tt.input, tt.index, tt.useNum)
			if len(got) != len(tt.want) {
				t.Fatalf("length mismatch: got %d (%v), want %d (%v)",
					len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("rcpt[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EML useHeaderEnvelope + numbering
// ---------------------------------------------------------------------------

// applyHeaderEnvelopeNumbering mirrors the exact logic used inside SendEMLRaw
// when useHeaderEnvelope == true. It extracts the bare From address, parses
// the To header into individual recipients, and applies numbering when enabled.
// Both usedFrom and usedTo strings are returned (rejoined for To).
func applyHeaderEnvelopeNumbering(emlFrom, emlTo string, numberingFrom, numberingTo bool, index int) (usedFrom, usedTo string) {
	usedFrom = extractAddr(emlFrom)
	usedTo = emlTo
	if numberingFrom {
		usedFrom = applyNumbering(usedFrom, index)
	}
	if numberingTo {
		parts := parseRecipients(usedTo)
		for i, r := range parts {
			bare := extractAddr(r)
			parts[i] = applyNumbering(bare, index)
		}
		usedTo = strings.Join(parts, ", ")
	}
	return usedFrom, usedTo
}

func TestEMLHeaderEnvelopeNumbering(t *testing.T) {
	tests := []struct {
		name          string
		emlFrom       string
		emlTo         string
		numberingFrom bool
		numberingTo   bool
		index         int
		wantFrom      string
		wantTo        string
	}{
		{
			name:     "no numbering — pass through",
			emlFrom:  "sender@example.com",
			emlTo:    "receiver@example.com",
			index:    0,
			wantFrom: "sender@example.com",
			wantTo:   "receiver@example.com",
		},
		{
			name:          "number From only",
			emlFrom:       "sender@example.com",
			emlTo:         "receiver@example.com",
			numberingFrom: true,
			index:         5,
			wantFrom:      "sender-5@example.com",
			wantTo:        "receiver@example.com",
		},
		{
			name:        "number To only",
			emlFrom:     "sender@example.com",
			emlTo:       "receiver@example.com",
			numberingTo: true,
			index:       3,
			wantFrom:    "sender@example.com",
			wantTo:      "receiver-3@example.com",
		},
		{
			name:          "number both",
			emlFrom:       "sender@example.com",
			emlTo:         "receiver@example.com",
			numberingFrom: true,
			numberingTo:   true,
			index:         7,
			wantFrom:      "sender-7@example.com",
			wantTo:        "receiver-7@example.com",
		},
		{
			name:          "display name in From header",
			emlFrom:       `"Alice Sender" <alice@example.com>`,
			emlTo:         "bob@example.com",
			numberingFrom: true,
			numberingTo:   true,
			index:         2,
			// extractAddr strips display name
			wantFrom: "alice-2@example.com",
			wantTo:   "bob-2@example.com",
		},
		{
			name:          "multiple recipients in To header",
			emlFrom:       "sender@example.com",
			emlTo:         "a@x.com, b@y.com, c@z.com",
			numberingFrom: false,
			numberingTo:   true,
			index:         4,
			wantFrom:      "sender@example.com",
			wantTo:        "a-4@x.com, b-4@y.com, c-4@z.com",
		},
		{
			name:          "multiple recipients with display names",
			emlFrom:       "sender@example.com",
			emlTo:         `"Alice" <alice@x.com>, "Bob" <bob@y.com>`,
			numberingTo:   true,
			index:         1,
			wantFrom:      "sender@example.com",
			wantTo:        "alice-1@x.com, bob-1@y.com",
		},
		{
			name:          "reported bug case — digits in domain preserved",
			emlFrom:       "sender@example.com",
			emlTo:         "aimaya@jiran.com, aimaya2@jiran.com, aimaya@jiran2.com",
			numberingTo:   true,
			index:         0,
			wantFrom:      "sender@example.com",
			wantTo:        "aimaya-0@jiran.com, aimaya2-0@jiran.com, aimaya-0@jiran2.com",
		},
		{
			name:          "semicolon separator in To header",
			emlFrom:       "sender@example.com",
			emlTo:         "a@x.com; b@y.com",
			numberingTo:   true,
			index:         9,
			wantFrom:      "sender@example.com",
			wantTo:        "a-9@x.com, b-9@y.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo := applyHeaderEnvelopeNumbering(tt.emlFrom, tt.emlTo, tt.numberingFrom, tt.numberingTo, tt.index)
			if gotFrom != tt.wantFrom {
				t.Errorf("From: got %q, want %q", gotFrom, tt.wantFrom)
			}
			if gotTo != tt.wantTo {
				t.Errorf("To: got %q, want %q", gotTo, tt.wantTo)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseRecipients standalone
// ---------------------------------------------------------------------------

func TestParseRecipients(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"single", "a@b.com", []string{"a@b.com"}},
		{"comma", "a@b.com,c@d.com", []string{"a@b.com", "c@d.com"}},
		{"comma with space", "a@b.com, c@d.com", []string{"a@b.com", "c@d.com"}},
		{"semicolon", "a@b.com;c@d.com", []string{"a@b.com", "c@d.com"}},
		{"mixed", "a@b.com, c@d.com; e@f.com", []string{"a@b.com", "c@d.com", "e@f.com"}},
		{"trailing separator", "a@b.com,", []string{"a@b.com"}},
		{"leading separator", ",a@b.com", []string{"a@b.com"}},
		{"double separator", "a@b.com,,c@d.com", []string{"a@b.com", "c@d.com"}},
		{"all empty", ",,,", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRecipients(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("length mismatch: got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
