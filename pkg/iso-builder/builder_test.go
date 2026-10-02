package isobuilder

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedConfigEmpty(t *testing.T) {
	_, err := loadEmbeddedConfig()
	if err == nil {
		t.Fatal("expected error for empty embed area, got nil")
	}
	if !strings.Contains(err.Error(), "no embedded configuration") {
		t.Errorf("unexpected error: %v", err)
	}
}
