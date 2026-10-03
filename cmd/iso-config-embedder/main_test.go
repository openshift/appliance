package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	isoconfigembedder "github.com/openshift/appliance/pkg/iso-config-embedder"
	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
)

func buildFakeBinary() []byte {
	prefix := []byte("ELF-FAKE-HEADER-PADDING-DATA")
	suffix := []byte("TRAILING-BINARY-DATA")
	buf := make([]byte, 0, len(prefix)+len(isobuilderconfig.ConfigStartMarker)+isobuilderconfig.ConfigEmbedSize+len(isobuilderconfig.ConfigEndMarker)+len(suffix))
	buf = append(buf, prefix...)
	buf = append(buf, isobuilderconfig.ConfigStartMarker...)
	buf = append(buf, make([]byte, isobuilderconfig.ConfigEmbedSize)...)
	buf = append(buf, isobuilderconfig.ConfigEndMarker...)
	buf = append(buf, suffix...)
	return buf
}

func writeFakeBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-iso-builder")
	if err := os.WriteFile(path, buildFakeBinary(), 0755); err != nil {
		t.Fatalf("writing fake binary: %v", err)
	}
	return path
}

func writeYAMLFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing YAML file: %v", err)
	}
	return path
}

func outputPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "output-binary")
}

func writeAuthFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing auth file: %v", err)
	}
	return path
}

func TestEmbedAndReadRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		wantKeys []string
	}{
		{
			name: "full config",
			yaml: `openshiftVersion: "4.22"
releaseImageURL: "quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64"
pullSecret: '{"auths":{}}'
sshKey:
  - ssh-rsa AAAA...
architecture: x86_64
proxy:
  httpProxy: http://proxy:8080
  httpsProxy: https://proxy:8443
  noProxy: localhost
additionalTrustBundle: |
  -----BEGIN CERTIFICATE-----
  MIID...
  -----END CERTIFICATE-----
additionalNTPServers:
  - ntp1.example.com
fips: true
olmOperators:
  - name: local-storage-operator
    version: "4.22"
    channel: stable
rendezvousIP: 192.168.1.10
additionalImages:
  - registry.example.com/app:v1
extraManifests:
  - name: custom.yaml
    content: "apiVersion: v1\nkind: ConfigMap"
`,
			wantKeys: []string{
				"openshiftVersion", "releaseImageURL", "pullSecret",
				"sshKey", "architecture", "proxy", "fips",
				"olmOperators", "rendezvousIP", "additionalImages",
			},
		},
		{
			name: "minimal config",
			yaml: `pullSecret: '{"auths":{}}'
`,
			wantKeys: []string{"pullSecret"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binaryPath := writeFakeBinary(t)
			configPath := writeYAMLFile(t, tc.yaml)
			out := outputPath(t)

			if err := isoconfigembedder.EmbedConfigFile(configPath, binaryPath, out, false); err != nil {
				t.Fatalf("isoconfigembedder.EmbedConfigFile: %v", err)
			}

			cfg, err := isobuilderconfig.ReadFromBinary(out)
			if err != nil {
				t.Fatalf("ReadFromBinary after embed: %v", err)
			}

			if cfg.PullSecret != `{"auths":{}}` {
				t.Errorf("pullSecret mismatch: got %q", cfg.PullSecret)
			}

			for _, key := range tc.wantKeys {
				if !strings.Contains(tc.yaml, key) {
					t.Errorf("test setup: expected YAML to contain %q", key)
				}
			}
		})
	}
}

func TestEmbedPullSecretFromRegistryAuthFile(t *testing.T) {
	secret := `{"auths":{"registry.example.com":{"auth":"dGVzdA=="}}}`

	cases := []struct {
		name       string
		yaml       string
		authFile   string
		setEnv     bool
		wantErr    string
		wantSecret string
	}{
		{
			name:       "env var provides pull secret when YAML omits it",
			yaml:       "openshiftVersion: \"4.22\"\n",
			setEnv:     true,
			wantSecret: secret,
		},
		{
			name:       "YAML pullSecret takes precedence over env var",
			yaml:       "pullSecret: '{\"auths\":{}}'\n",
			setEnv:     true,
			wantSecret: `{"auths":{}}`,
		},
		{
			name:    "env var points to nonexistent file",
			yaml:    "openshiftVersion: \"4.22\"\n",
			setEnv:  true,
			wantErr: "reading pull secret from PULL_SECRET_FILE",
		},
		{
			name:       "env var not set and no pull secret in YAML",
			yaml:       "openshiftVersion: \"4.22\"\n",
			setEnv:     false,
			wantSecret: "",
		},
		{
			name:       "trailing whitespace in auth file is trimmed",
			yaml:       "openshiftVersion: \"4.22\"\n",
			setEnv:     true,
			wantSecret: secret,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binaryPath := writeFakeBinary(t)
			configPath := writeYAMLFile(t, tc.yaml)
			out := outputPath(t)

			if tc.setEnv {
				switch tc.name {
				case "env var points to nonexistent file":
					t.Setenv("PULL_SECRET_FILE", "/nonexistent/auth.json")
				case "trailing whitespace in auth file is trimmed":
					t.Setenv("PULL_SECRET_FILE", writeAuthFile(t, secret+"\n\n"))
				default:
					t.Setenv("PULL_SECRET_FILE", writeAuthFile(t, secret))
				}
			}

			err := isoconfigembedder.EmbedConfigFile(configPath, binaryPath, out, false)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("isoconfigembedder.EmbedConfigFile: %v", err)
			}

			if tc.wantSecret == "" {
				return
			}

			cfg, err := isobuilderconfig.ReadFromBinary(out)
			if err != nil {
				t.Fatalf("ReadFromBinary: %v", err)
			}
			if cfg.PullSecret != tc.wantSecret {
				t.Errorf("pullSecret: got %q, want %q", cfg.PullSecret, tc.wantSecret)
			}
		})
	}
}

func TestShowOutputsYAML(t *testing.T) {
	binaryPath := writeFakeBinary(t)
	configPath := writeYAMLFile(t, `openshiftVersion: "4.22"
pullSecret: '{"auths":{}}'
architecture: x86_64
`)
	out := outputPath(t)

	if err := isoconfigembedder.EmbedConfigFile(configPath, binaryPath, out, false); err != nil {
		t.Fatalf("isoconfigembedder.EmbedConfigFile: %v", err)
	}

	cfg, err := isobuilderconfig.ReadFromBinary(out)
	if err != nil {
		t.Fatalf("ReadFromBinary: %v", err)
	}

	if cfg.OpenshiftVersion != "4.22" {
		t.Errorf("openshiftVersion: got %q, want %q", cfg.OpenshiftVersion, "4.22")
	}
	if cfg.Architecture != "x86_64" {
		t.Errorf("architecture: got %q, want %q", cfg.Architecture, "x86_64")
	}
	if cfg.PullSecret != `{"auths":{}}` {
		t.Errorf("pullSecret: got %q", cfg.PullSecret)
	}
}


func TestEmbedInvalidYAML(t *testing.T) {
	binaryPath := writeFakeBinary(t)
	configPath := writeYAMLFile(t, `{invalid: yaml: [`)

	err := isoconfigembedder.EmbedConfigFile(configPath, binaryPath, outputPath(t), false)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "parsing YAML") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmbedMissingConfigFile(t *testing.T) {
	binaryPath := writeFakeBinary(t)

	err := isoconfigembedder.EmbedConfigFile("/nonexistent/config.yaml", binaryPath, outputPath(t), false)
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
	if !strings.Contains(err.Error(), "reading config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmbedMissingBinary(t *testing.T) {
	configPath := writeYAMLFile(t, `pullSecret: '{"auths":{}}'`)

	err := isoconfigembedder.EmbedConfigFile(configPath, "/nonexistent/binary", outputPath(t), false)
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
	if !strings.Contains(err.Error(), "reading binary") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestShowNoEmbeddedConfig(t *testing.T) {
	binaryPath := writeFakeBinary(t)

	err := isoconfigembedder.ShowConfigFromBinary(binaryPath)
	if err == nil {
		t.Fatal("expected error for empty embed area, got nil")
	}
	if !strings.Contains(err.Error(), "reading embedded config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestShowMissingBinary(t *testing.T) {
	err := isoconfigembedder.ShowConfigFromBinary("/nonexistent/binary")
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
	if !strings.Contains(err.Error(), "reading embedded config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmbedStrictYAMLRejectsUnknownFields(t *testing.T) {
	binaryPath := writeFakeBinary(t)
	configPath := writeYAMLFile(t, `pullSecret: '{"auths":{}}'
unknownField: should-be-rejected
`)

	err := isoconfigembedder.EmbedConfigFile(configPath, binaryPath, outputPath(t), false)
	if err == nil {
		t.Fatal("expected error for unknown YAML field, got nil")
	}
	if !strings.Contains(err.Error(), "parsing YAML") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmbedForceOverwrite(t *testing.T) {
	binaryPath := writeFakeBinary(t)
	first := writeYAMLFile(t, `pullSecret: '{"auths":{"first":{}}}'`)
	embedded := outputPath(t)

	if err := isoconfigembedder.EmbedConfigFile(first, binaryPath, embedded, false); err != nil {
		t.Fatalf("first embed: %v", err)
	}

	second := writeYAMLFile(t, `pullSecret: '{"auths":{"second":{}}}'`)
	out2 := filepath.Join(t.TempDir(), "output2")

	err := isoconfigembedder.EmbedConfigFile(second, embedded, out2, false)
	if err == nil {
		t.Fatal("expected error when overwriting without --force")
	}
	if !strings.Contains(err.Error(), "already contains") {
		t.Errorf("unexpected error: %v", err)
	}

	if err := isoconfigembedder.EmbedConfigFile(second, embedded, out2, true); err != nil {
		t.Fatalf("embed with --force: %v", err)
	}

	cfg, err := isobuilderconfig.ReadFromBinary(out2)
	if err != nil {
		t.Fatalf("ReadFromBinary: %v", err)
	}
	if cfg.PullSecret != `{"auths":{"second":{}}}` {
		t.Errorf("expected second config, got pullSecret %q", cfg.PullSecret)
	}
}

func TestEmbedDoesNotModifySource(t *testing.T) {
	binaryPath := writeFakeBinary(t)
	configPath := writeYAMLFile(t, `pullSecret: '{"auths":{}}'`)
	out := outputPath(t)

	if err := isoconfigembedder.EmbedConfigFile(configPath, binaryPath, out, false); err != nil {
		t.Fatalf("embed: %v", err)
	}

	if _, err := isobuilderconfig.ReadFromBinary(binaryPath); err == nil {
		t.Error("source binary should not contain an embedded config")
	}

	cfg, err := isobuilderconfig.ReadFromBinary(out)
	if err != nil {
		t.Fatalf("ReadFromBinary on output: %v", err)
	}
	if cfg.PullSecret != `{"auths":{}}` {
		t.Errorf("pullSecret mismatch: got %q", cfg.PullSecret)
	}
}

func TestEmbedFromStdin(t *testing.T) {
	binaryPath := writeFakeBinary(t)
	out := outputPath(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		_, _ = w.WriteString("pullSecret: '{\"auths\":{}}'\n")
		_ = w.Close()
	}()

	if err := isoconfigembedder.EmbedConfigFile("", binaryPath, out, false); err != nil {
		t.Fatalf("isoconfigembedder.EmbedConfigFile from stdin: %v", err)
	}

	cfg, err := isobuilderconfig.ReadFromBinary(out)
	if err != nil {
		t.Fatalf("ReadFromBinary: %v", err)
	}
	if cfg.PullSecret != `{"auths":{}}` {
		t.Errorf("pullSecret mismatch: got %q", cfg.PullSecret)
	}
}
