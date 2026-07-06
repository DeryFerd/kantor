package operational

import (
	"encoding/json"
	"testing"
)

func TestFirefoxManifestRewritesBackground(t *testing.T) {
	raw := []byte(`{"manifest_version":3,"version":"2.0.1","background":{"service_worker":"background.js"},"permissions":["idle"]}`)
	out, err := firefoxManifest(raw)
	if err != nil {
		t.Fatalf("firefoxManifest: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("output not valid json: %v", err)
	}
	bg, ok := m["background"].(map[string]any)
	if !ok {
		t.Fatalf("background missing/invalid: %v", m["background"])
	}
	if _, hasSW := bg["service_worker"]; hasSW {
		t.Error("firefox manifest must not keep background.service_worker (Chrome-only)")
	}
	scripts, ok := bg["scripts"].([]any)
	if !ok || len(scripts) != 1 || scripts[0] != "background.js" {
		t.Errorf("expected background.scripts=[background.js], got %v", bg["scripts"])
	}
	if m["version"] != "2.0.1" {
		t.Errorf("version must be preserved, got %v", m["version"])
	}
	if m["permissions"] == nil {
		t.Error("other keys must be preserved")
	}
}

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
