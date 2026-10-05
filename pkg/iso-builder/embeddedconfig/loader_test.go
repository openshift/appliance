package embeddedconfig

import (
	"strings"
	"testing"
)

func TestLoadConfigEmpty(t *testing.T) {
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for empty embed area, got nil")
	}
	if !strings.Contains(err.Error(), "no embedded configuration") {
		t.Errorf("unexpected error: %v", err)
	}
}
