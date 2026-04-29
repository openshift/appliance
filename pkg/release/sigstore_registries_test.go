package release

import (
	"strings"
	"testing"

	"github.com/openshift/appliance/pkg/types"
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

func TestAdditionalImageRegistryHosts(t *testing.T) {
	t.Parallel()
	images := []types.Image{
		{Name: "registry.connect.redhat.com/partner/foo@sha256:deadbeef"},
		{Name: "docker://quay.io/example/sidecar:latest"},
		{Name: "registry.connect.redhat.com/partner/bar:1.0"},
	}
	got := additionalImageRegistryHosts(&images)
	want := []string{"quay.io", "registry.connect.redhat.com"}
	if len(got) != len(want) {
		t.Fatalf("additionalImageRegistryHosts() returned %d hosts, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("additionalImageRegistryHosts() host at index %d = %q, want %q", i, got[i], want[i])
		}
	}

	if hosts := additionalImageRegistryHosts(nil); len(hosts) != 0 {
		t.Fatalf("expected empty list for nil images, got %v", hosts)
	}

	var empty []types.Image
	if hosts := additionalImageRegistryHosts(&empty); len(hosts) != 0 {
		t.Fatalf("expected empty list for empty images, got %v", hosts)
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
