package ocmirror

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/openshift/appliance/pkg/tools"
)

func TestExtractOcMirrorFromArchive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a tar.gz with oc-mirror binary
	archivePath := filepath.Join(tmpDir, "oc-mirror.tar.gz")
	createTestArchive(t, archivePath, ocMirrorBinaryName, "test binary content")

	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	result, err := tools.ExtractBinaryFromArchive(archivePath, destDir, ocMirrorBinaryName)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedPath := filepath.Join(destDir, ocMirrorBinaryName)
	if result != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result)
	}

	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(content) != "test binary content" {
		t.Errorf("expected 'test binary content', got '%s'", string(content))
	}
}

func TestExtractOcMirrorNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a tar.gz without oc-mirror binary
	archivePath := filepath.Join(tmpDir, "oc-mirror.tar.gz")
	createTestArchive(t, archivePath, "other-file", "some content")

	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, err := tools.ExtractBinaryFromArchive(archivePath, destDir, ocMirrorBinaryName)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// createTestArchive creates a simple tar.gz file for testing
func createTestArchive(t *testing.T, archivePath, filename, content string) {
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer func() { _ = f.Close() }()

	gw := gzip.NewWriter(f)
	defer func() { _ = gw.Close() }()

	tw := tar.NewWriter(gw)
	defer func() { _ = tw.Close() }()

	hdr := &tar.Header{
		Name:     filename,
		Mode:     0644,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}
	if _, err := io.WriteString(tw, content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
}

func TestNewMirror(t *testing.T) {
	config := MirrorConfig{
		ToolConfig: tools.ToolConfig{
			ReleaseVersion: "4.22.19",
			ToolsDir:       "/tmp/tools",
			SupportedArch:  "x86_64",
		},
	}
	m := NewMirror(config)
	if m == nil {
		t.Fatal("expected non-nil Mirror")
	}
	if m.config != config {
		t.Errorf("expected config to be set correctly")
	}
}

func TestAcquireWithExistingBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing binary
	binPath := filepath.Join(tmpDir, ocMirrorBinaryName)
	f, err := os.Create(binPath)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	_, err = f.WriteString("fake binary")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	_ = f.Close()

	m := NewMirror(MirrorConfig{
		ToolConfig: tools.ToolConfig{
			ReleaseVersion: "4.22.19",
			ToolsDir:       tmpDir,
			SupportedArch:  "x86_64",
		},
	})

	result, err := m.Acquire()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result.Path != binPath {
		t.Errorf("expected path %s, got %s", binPath, result.Path)
	}
	if result.FromSystem {
		t.Error("expected FromSystem to be false")
	}
}

