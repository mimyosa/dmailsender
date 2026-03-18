package core

import "testing"

func TestApplyNumbering(t *testing.T) {
	tests := []struct {
		input string
		index int
		want  string
	}{
		{"user001@example.com", 5, "user005@example.com"},
		{"user@example.com", 3, "user3@example.com"},
		{"test100@example.com", 1, "test001@example.com"},
		{"user001@example.com", 999, "user999@example.com"},
		{"sender01@test.org", 0, "sender00@test.org"},
		{"a1@b.com", 99, "a99@b.com"},
		{"no-digits@example.com", 42, "no-digits42@example.com"},
	}

	for _, tt := range tests {
		got := applyNumbering(tt.input, tt.index)
		if got != tt.want {
			t.Errorf("applyNumbering(%q, %d) = %q, want %q", tt.input, tt.index, got, tt.want)
		}
	}
}

func TestApplyNumberingSubject(t *testing.T) {
	tests := []struct {
		input string
		index int
		want  string
	}{
		{"Test Message 001", 5, "Test Message 005"},
		{"Test Message", 3, "Test Message3"},
		{"Subject100", 42, "Subject042"},
	}

	for _, tt := range tests {
		got := applyNumberingSubject(tt.input, tt.index)
		if got != tt.want {
			t.Errorf("applyNumberingSubject(%q, %d) = %q, want %q", tt.input, tt.index, got, tt.want)
		}
	}
}
