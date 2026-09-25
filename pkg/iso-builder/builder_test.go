package isobuilder

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedConfigEmpty(t *testing.T) {
	b := NewBuilder(t.TempDir())
	_, err := b.loadEmbeddedConfig()
	if err == nil {
		t.Fatal("expected error for empty embed area, got nil")
	}
	if !strings.Contains(err.Error(), "no embedded configuration") {
		t.Errorf("unexpected error: %v", err)
	}
}
