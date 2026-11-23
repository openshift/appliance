package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetDownloadURL(t *testing.T) {
	const ocMirrorURL = "https://mirror.openshift.com/pub/cgw/oc-mirror/latest/oc-mirror-rhel9-linux-amd64.tar.gz"

	cases := []struct {
		name           string
		toolName       string
		releaseVersion string
		platform       string
		wantErr        string
		checkContains  []string
	}{
		{
			name:           "oc stable release",
			toolName:       "oc",
			releaseVersion: "4.22.3",
			platform:       platformLinuxAmd64,
			checkContains:  []string{"mirror.openshift.com", "4.22.3", "ocp", "openshift-client"},
		},
		{
			name:           "oc dev preview",
			toolName:       "oc",
			releaseVersion: "4.22.0-ec.5",
			platform:       platformLinuxAmd64,
			checkContains:  []string{"ocp-dev-preview", "4.22.0-ec.5"},
		},
		{
			name:           "oc-mirror OpenShift 4",
			toolName:       "oc-mirror",
			releaseVersion: "4.22.3",
			platform:       platformLinuxAmd64,
			checkContains:  []string{ocMirrorURL},
		},
		{
			name:           "oc OpenShift 5",
			toolName:       "oc",
			releaseVersion: "5.1.0",
			platform:       platformLinuxAmd64,
			checkContains:  []string{"pub/openshift-v5/x86_64/clients/ocp/5.1.0/openshift-client-linux.tar.gz"},
		},
		{
			name:           "oc-mirror OpenShift 5",
			toolName:       "oc-mirror",
			releaseVersion: "5.0.0",
			platform:       platformLinuxAmd64,
			checkContains:  []string{ocMirrorURL},
		},
		{
			name:           "oc OpenShift 5 on mac",
			toolName:       "oc",
			releaseVersion: "5.0.0",
			platform:       platformDarwinArm64,
			checkContains:  []string{"pub/openshift-v5/x86_64/clients/ocp/5.0.0/openshift-client-mac-arm64.tar.gz"},
		},
		{
			name:           "oc-mirror on mac",
			toolName:       "oc-mirror",
			releaseVersion: "4.22.3",
			platform:       platformDarwinArm64,
			wantErr:        "oc-mirror is not published for darwin-arm64, provide it on PATH",
		},
		{
			name:           "oc on unsupported platform",
			toolName:       "oc",
			releaseVersion: "4.22.3",
			platform:       "linux-s390x",
			wantErr:        "unsupported platform: linux-s390x",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := getDownloadURL(tc.toolName, tc.releaseVersion, "x86_64", tc.platform)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, substr := range tc.checkContains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected URL to contain %q, got %s", substr, result)
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
