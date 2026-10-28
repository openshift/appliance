package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)



func TestGetDownloadURL(t *testing.T) {
	cases := []struct {
		name           string
		toolName       string
		releaseVersion string
		supportedArch  string
		wantErr        bool
		checkContains  []string
	}{
		{
			name:           "oc stable release",
			toolName:       "oc",
			releaseVersion: "4.22.3",
			supportedArch:  "x86_64",
			checkContains:  []string{"mirror.openshift.com", "4.22.3", "ocp", "openshift-client"},
		},
		{
			name:           "oc dev preview",
			toolName:       "oc",
			releaseVersion: "4.22.0-ec.5",
			supportedArch:  "x86_64",
			checkContains:  []string{"ocp-dev-preview", "4.22.0-ec.5"},
		},
		{
			name:           "oc-mirror stable",
			toolName:       "oc-mirror",
			releaseVersion: "4.22.3",
			supportedArch:  "x86_64",
			checkContains:  []string{"oc-mirror", "4.22.3"},
		},
		{
			name:          "unknown tool",
			toolName:      "unknown",
			releaseVersion: "4.22.3",
			supportedArch:  "x86_64",
			wantErr:       true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GetDownloadURL(tc.toolName, tc.releaseVersion, tc.supportedArch)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				for _, substr := range tc.checkContains {
					if !strings.Contains(result, substr) {
						t.Errorf("expected URL to contain %q, got %s", substr, result)
					}
				}
			}
		})
	}
}

func TestCreateToolsDir(t *testing.T) {
	tmpDir := t.TempDir()
	toolsDir := filepath.Join(tmpDir, "tools")

	err := CreateToolsDir(toolsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(toolsDir)
	if err != nil {
		t.Fatalf("failed to stat tools dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected tools dir to be a directory")
	}
}

func TestFindExistingBinary(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(string)
		wantFound bool
	}{
		{
			name: "binary exists with content",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "testbin"), []byte("content"), 0755)
			},
			wantFound: true,
		},
		{
			name: "binary doesn't exist",
			setup: func(dir string) {
			},
			wantFound: false,
		},
		{
			name: "binary is empty",
			setup: func(dir string) {
				_, _ = os.Create(filepath.Join(dir, "testbin"))
			},
			wantFound: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tc.setup(tmpDir)

			result := FindExistingBinary(tmpDir, "testbin")
			if tc.wantFound {
				expected := filepath.Join(tmpDir, "testbin")
				if result != expected {
					t.Errorf("expected %s, got %s", expected, result)
				}
			} else {
				if result != "" {
					t.Errorf("expected empty, got %s", result)
				}
			}
		})
	}
}
