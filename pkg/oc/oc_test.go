package oc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openshift/appliance/pkg/tools"
)

func TestAcquireReusesExistingBinary(t *testing.T) {
	toolsDir := t.TempDir()

	ocPath := filepath.Join(toolsDir, "oc")
	if err := os.WriteFile(ocPath, []byte("fake-oc-binary"), 0755); err != nil {
		t.Fatalf("failed to create fake oc binary: %v", err)
	}

	c := NewClient(ClientConfig{
		ToolConfig: tools.ToolConfig{
			ReleaseVersion: "4.22.3",
			ToolsDir:       toolsDir,
			SupportedArch:  "x86_64",
		},
	})

	result, err := c.Acquire()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != ocPath {
		t.Errorf("expected existing path %s, got %s", ocPath, result.Path)
	}
	if result.FromSystem {
		t.Error("expected FromSystem to be false for existing binary")
	}
}

