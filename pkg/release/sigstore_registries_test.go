package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageReferenceRegistryHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"registry.connect.redhat.com/foo/bar@sha256:abc", "registry.connect.redhat.com"},
		{"docker://registry.connect.redhat.com/foo/bar:latest", "registry.connect.redhat.com"},
		{"quay.io/foo/bar", "quay.io"},
		{"registry.redhat.io/openshift/release", "registry.redhat.io"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := imageReferenceRegistryHost(tc.in); got != tc.want {
				t.Fatalf("imageReferenceRegistryHost(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeRegistryHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"registry.connect.redhat.com", "registry.connect.redhat.com"},
		{" registry.connect.redhat.com ", "registry.connect.redhat.com"},
		{"registry.connect.redhat.com:443", "registry.connect.redhat.com"},
		{"registry.connect.redhat.com/partner/foo:latest", "registry.connect.redhat.com"},
		{"docker://quay.io/example:latest", "quay.io"},
		{"", ""},
		{"invalid/host/extra", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := normalizeRegistryHost(tc.in); got != tc.want {
				t.Fatalf("normalizeRegistryHost(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDisableSigstoreRegistryHosts(t *testing.T) {
	t.Parallel()
	registries := []string{
		"registry.connect.redhat.com",
		" registry.connect.redhat.com ",
		"quay.io",
		"quay.io/example:latest",
		"",
	}
	got := disableSigstoreRegistryHosts(&registries)
	want := []string{"quay.io", "registry.connect.redhat.com"}
	if len(got) != len(want) {
		t.Fatalf("disableSigstoreRegistryHosts() returned %d hosts, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("disableSigstoreRegistryHosts() host at index %d = %q, want %q", i, got[i], want[i])
		}
	}

	if hosts := disableSigstoreRegistryHosts(nil); len(hosts) != 0 {
		t.Fatalf("expected empty list for nil registries, got %v", hosts)
	}

	var empty []string
	if hosts := disableSigstoreRegistryHosts(&empty); len(hosts) != 0 {
		t.Fatalf("expected empty list for empty registries, got %v", hosts)
	}
}

func TestBuildDisableSigstoreRegistriesConfig(t *testing.T) {
	t.Parallel()

	config := string(buildDisableSigstoreRegistriesConfig([]string{"quay.io", "registry.connect.redhat.com"}))

	if !strings.Contains(config, "docker:\n") {
		t.Fatal("expected docker section in generated config")
	}
	if !strings.Contains(config, "  quay.io:\n    use-sigstore-attachments: false\n") {
		t.Fatal("expected quay.io entry in generated config")
	}
	if !strings.Contains(config, "  registry.connect.redhat.com:\n    use-sigstore-attachments: false\n") {
		t.Fatal("expected registry.connect.redhat.com entry in generated config")
	}
}

func TestFilterHostsWithoutExistingRegistriesDConfig(t *testing.T) {
	t.Parallel()

	registriesDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(registriesDir, "registry.redhat.io.yaml"), []byte("docker:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	hosts := []string{"registry.redhat.io", "registry.connect.redhat.com", "quay.io"}
	got := filterHostsWithoutExistingRegistriesDConfig(hosts, registriesDir, os.Stat)
	want := []string{"registry.connect.redhat.com", "quay.io"}
	if len(got) != len(want) {
		t.Fatalf("filterHostsWithoutExistingRegistriesDConfig() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("host at index %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDisableSigstoreRegistryConfigPath(t *testing.T) {
	t.Parallel()
	want := filepath.Join(registriesDDir, "registry.connect.redhat.com.yaml")
	if got := disableSigstoreRegistryConfigPath("registry.connect.redhat.com"); got != want {
		t.Fatalf("disableSigstoreRegistryConfigPath() = %q, want %q", got, want)
	}
}
