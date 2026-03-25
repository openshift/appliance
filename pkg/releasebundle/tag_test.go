package releasebundle

import "testing"

func TestTag(t *testing.T) {
	tests := []struct {
		version  string
		wantTag  string
	}{
		{"4.21.0-ec.3-x86_64", "ocp-release-bundle-4.21.0-ec.3-x86_64"},
		{"4.20.5-x86_64", "ocp-release-bundle-4.20.5-x86_64"},
		{"4.14.0-0.nightly-2025-11-23-025204", "ocp-release-bundle-4.14.0-0.nightly-2025-11-23-025204"},
		{"4.22.0-0.ci-2026-02-09-204741-test-ci-op-phx0mrh8-latest", "ocp-release-bundle-4.22.0-0.ci-2026-02-09-204741-test-ci-op-phx0"},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := Tag(tt.version); got != tt.wantTag {
				t.Errorf("Tag(%q) = %q, want %q", tt.version, got, tt.wantTag)
			}
		})
	}
}
