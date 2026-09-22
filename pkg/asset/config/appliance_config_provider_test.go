package config

import (
	"testing"
)

func TestApplianceConfigProviderDefaults(t *testing.T) {
	provider := &ApplianceConfigProvider{}

	if err := provider.Generate(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Config != nil {
		t.Error("expected Config to be nil by default")
	}

	found, err := provider.Load(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected Load to return false")
	}

	if len(provider.Files()) != 0 {
		t.Error("expected no files")
	}
}
