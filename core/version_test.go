package core

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int // >0, 0, <0
	}{
		{"1.0", "1.0", 0},
		{"1.1", "1.0", 1},
		{"1.0", "1.1", -1},
		{"2.0", "1.9", 1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.0", "1.0.0", 0},
		{"1.2.3", "1.2.3", 0},
		{"10.0", "9.0", 1},
	}

	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if (tt.want > 0 && got <= 0) || (tt.want < 0 && got >= 0) || (tt.want == 0 && got != 0) {
			t.Errorf("compareVersions(%q, %q) = %d, want sign %d", tt.a, tt.b, got, tt.want)
		}
	}
}
