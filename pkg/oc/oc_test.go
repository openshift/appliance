package oc

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDownloadURL(t *testing.T) {
	cases := []struct {
		name           string
		releaseVersion string
		goos           string
		goarch         string
		expectedURL    string
	}{
		{
			name:           "stable linux amd64",
			releaseVersion: "4.22.3",
			goos:           "linux",
			goarch:         "amd64",
			expectedURL:    "https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.22.3/openshift-client-linux.tar.gz",
		},
		{
			name:           "EC preview linux amd64",
			releaseVersion: "4.22.0-ec.5",
			goos:           "linux",
			goarch:         "amd64",
			expectedURL:    "https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp-dev-preview/4.22.0-ec.5/openshift-client-linux.tar.gz",
		},
		{
			name:           "stable darwin arm64",
			releaseVersion: "4.22.3",
			goos:           "darwin",
			goarch:         "arm64",
			expectedURL:    "https://mirror.openshift.com/pub/openshift-v4/aarch64/clients/ocp/4.22.3/openshift-client-mac-arm64.tar.gz",
		},
		{
			name:           "stable linux arm64",
			releaseVersion: "4.22.3",
			goos:           "linux",
			goarch:         "arm64",
			expectedURL:    "https://mirror.openshift.com/pub/openshift-v4/aarch64/clients/ocp/4.22.3/openshift-client-linux-arm64.tar.gz",
		},
		{
			name:           "stable darwin amd64",
			releaseVersion: "4.22.3",
			goos:           "darwin",
			goarch:         "amd64",
			expectedURL:    "https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.22.3/openshift-client-mac.tar.gz",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origOS, origArch := goOS, goArch
			goOS, goArch = tc.goos, tc.goarch
			defer func() { goOS, goArch = origOS, origArch }()

			c := NewClient(ClientConfig{ReleaseVersion: tc.releaseVersion})
			url, err := c.downloadURL()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if url != tc.expectedURL {
				t.Errorf("expected %s, got %s", tc.expectedURL, url)
			}
		})
	}
}

func TestDownloadURLUnsupportedPlatform(t *testing.T) {
	cases := []struct {
		name   string
		goos   string
		goarch string
	}{
		{
			name:   "windows amd64",
			goos:   "windows",
			goarch: "amd64",
		},
		{
			name:   "linux unsupported arch",
			goos:   "linux",
			goarch: "ppc64le",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origOS, origArch := goOS, goArch
			goOS, goArch = tc.goos, tc.goarch
			defer func() { goOS, goArch = origOS, origArch }()

			c := NewClient(ClientConfig{ReleaseVersion: "4.22.3"})
			_, err := c.downloadURL()
			if err == nil {
				t.Fatal("expected error for unsupported platform")
			}
		})
	}
}

func TestAcquireCacheHit(t *testing.T) {
	cacheDir := t.TempDir()

	ocPath := filepath.Join(cacheDir, "oc")
	if err := os.WriteFile(ocPath, []byte("fake-oc-binary"), 0755); err != nil {
		t.Fatalf("failed to create fake oc binary: %v", err)
	}

	c := NewClient(ClientConfig{
		ReleaseVersion: "4.22.3",
		CacheDir:       cacheDir,
	})

	result, err := c.Acquire()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != ocPath {
		t.Errorf("expected cached path %s, got %s", ocPath, result.Path)
	}
	if result.FromSystem {
		t.Error("expected FromSystem to be false for cached binary")
	}
}

func TestAcquireSystemFallback(t *testing.T) {
	cacheDir := t.TempDir()
	fakeOcDir := t.TempDir()
	fakeOcPath := filepath.Join(fakeOcDir, "oc")
	if err := os.WriteFile(fakeOcPath, []byte("fake-system-oc"), 0755); err != nil {
		t.Fatalf("failed to create fake system oc: %v", err)
	}

	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "oc" {
			return fakeOcPath, nil
		}
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}
	defer func() { lookPath = origLookPath }()

	c := NewClient(ClientConfig{
		ReleaseVersion: "99.0.0",
		CacheDir:       cacheDir,
	})

	result, err := c.Acquire()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != fakeOcPath {
		t.Errorf("expected system path %s, got %s", fakeOcPath, result.Path)
	}
	if !result.FromSystem {
		t.Error("expected FromSystem to be true for system fallback")
	}
}

func TestAcquireNoDownloadNoSystem(t *testing.T) {
	cacheDir := t.TempDir()

	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}
	defer func() { lookPath = origLookPath }()

	c := NewClient(ClientConfig{
		ReleaseVersion: "99.0.0",
		CacheDir:       cacheDir,
	})

	_, err := c.Acquire()
	if err == nil {
		t.Fatal("expected error when download and system lookup both fail")
	}
}
