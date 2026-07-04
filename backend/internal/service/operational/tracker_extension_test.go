package operational

import "testing"

func TestIsExtensionVersionOlder(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"1.0.0", "2.0.0", true},
		{"1.9.0", "1.10.0", true}, // numeric, not lexical
		{"2.0.0", "2.0.0", false},
		{"2.0.1", "2.0.0", false},
		{"1.0", "1.0.0", false},
		{"", "2.0.0", true},
		{"2.0.0", "", false},
	}
	for _, tc := range tests {
		if got := isExtensionVersionOlder(tc.current, tc.latest); got != tc.want {
			t.Errorf("isExtensionVersionOlder(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}
