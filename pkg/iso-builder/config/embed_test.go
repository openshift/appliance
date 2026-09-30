package isobuilder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	cases := []struct {
		name   string
		config Config
	}{
		{
			name: "full config",
			config: Config{
				OpenshiftVersion:      "4.22",
				ReleaseImageURL:       "quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64",
				PullSecret:            `{"auths":{}}`,
				SSHKey:                []string{"ssh-rsa AAAA..."},
				Architecture:          "x86_64",
				Proxy:                 &Proxy{HTTPProxy: "http://proxy:8080", HTTPSProxy: "https://proxy:8443", NoProxy: "localhost"},
				AdditionalTrustBundle: "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
				AdditionalNTPServers:  []string{"ntp1.example.com"},
				FIPS:                  true,
				OLMOperators:          []OLMOperator{{Name: "local-storage-operator", Version: "4.22", Channel: "stable"}},
				RendezvousIP:          "192.168.1.10",
				NetworkConfig:         []json.RawMessage{json.RawMessage(`{"interfaces":[]}`)},
				AdditionalImages:      []string{"registry.example.com/app:v1"},
				ExtraManifests:        []ExtraManifest{{Name: "custom.yaml", Content: "apiVersion: v1\nkind: ConfigMap"}},
			},
		},
		{
			name: "minimal config",
			config: Config{
				PullSecret: `{"auths":{}}`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := Encode(&tc.config)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			decoded, err := Decode(encoded)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			got, _ := json.Marshal(decoded)
			want, _ := json.Marshal(&tc.config)
			if string(got) != string(want) {
				t.Errorf("round-trip mismatch\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	cases := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "invalid base64",
			data:    []byte("not-valid-base64!@#$"),
			wantErr: "decoding base64",
		},
		{
			name:    "valid base64 but invalid JSON",
			data:    []byte("bm90LWpzb24="), // "not-json" in base64
			wantErr: "unmarshaling config",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(tc.data)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestReadWriteBinary(t *testing.T) {
	cfg := &Config{
		OpenshiftVersion: "4.22",
		PullSecret:       `{"auths":{}}`,
		Architecture:     "x86_64",
		OLMOperators:     []OLMOperator{{Name: "lvms-operator", Channel: "stable"}},
	}

	bin := buildFakeBinary()
	path := filepath.Join(t.TempDir(), "fake-binary")
	if err := os.WriteFile(path, bin, 0755); err != nil {
		t.Fatalf("writing fake binary: %v", err)
	}

	if err := WriteToBinary(path, cfg); err != nil {
		t.Fatalf("WriteToBinary: %v", err)
	}

	got, err := ReadFromBinary(path)
	if err != nil {
		t.Fatalf("ReadFromBinary: %v", err)
	}

	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(cfg)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("round-trip mismatch\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

func TestReadBinaryNoMarkers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-markers")
	if err := os.WriteFile(path, []byte("just some random binary content"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err := ReadFromBinary(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "marker not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReadBinaryEmptyArea(t *testing.T) {
	bin := buildFakeBinary()
	path := filepath.Join(t.TempDir(), "empty-area")
	if err := os.WriteFile(path, bin, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := ReadFromBinary(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no embedded configuration") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWriteBinaryTooLarge(t *testing.T) {
	tinyArea := make([]byte, 0, len(ConfigStartMarker)+16+len(ConfigEndMarker))
	tinyArea = append(tinyArea, ConfigStartMarker...)
	tinyArea = append(tinyArea, make([]byte, 16)...)
	tinyArea = append(tinyArea, ConfigEndMarker...)

	path := filepath.Join(t.TempDir(), "tiny-area")
	if err := os.WriteFile(path, tinyArea, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		PullSecret:       strings.Repeat("x", 100),
		OpenshiftVersion: "4.22",
	}
	err := WriteToBinary(path, cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds embed area") {
		t.Errorf("unexpected error: %v", err)
	}
}

func buildFakeBinary() []byte {
	prefix := []byte("ELF-FAKE-HEADER-PADDING-DATA")
	suffix := []byte("TRAILING-BINARY-DATA")
	buf := make([]byte, 0, len(prefix)+len(ConfigStartMarker)+ConfigEmbedSize+len(ConfigEndMarker)+len(suffix))
	buf = append(buf, prefix...)
	buf = append(buf, ConfigStartMarker...)
	buf = append(buf, make([]byte, ConfigEmbedSize)...)
	buf = append(buf, ConfigEndMarker...)
	buf = append(buf, suffix...)
	return buf
}
