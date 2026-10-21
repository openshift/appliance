package manifests

import (
	"context"
	"testing"

	"github.com/go-openapi/swag/conv"
	"sigs.k8s.io/yaml"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/types"
	"github.com/openshift/installer/pkg/asset"
	agenttypes "github.com/openshift/installer/pkg/types/agent"
)

func TestAgentConfigGenerate(t *testing.T) {
	cases := []struct {
		name         string
		rendezvousIP *string
		wantFile     bool
		wantIP       string
	}{
		{
			name:     "nil rendezvous IP produces no file",
			wantFile: false,
		},
		{
			name:         "set rendezvous IP produces agent-config.yaml",
			rendezvousIP: conv.Pointer("192.168.1.10"),
			wantFile:     true,
			wantIP:       "192.168.1.10",
		},
		{
			name:         "IPv6 address produces agent-config.yaml",
			rendezvousIP: conv.Pointer("fd00::1"),
			wantFile:     true,
			wantIP:       "fd00::1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appCfg := &config.ApplianceConfig{
				Config: &types.ApplianceConfig{
					ApplianceConfigExt: types.ApplianceConfigExt{
						RendezvousIP: tc.rendezvousIP,
					},
				},
			}

			parents := asset.Parents{}
			parents.Add(appCfg)

			a := &AgentConfig{}
			if err := a.Generate(context.Background(), parents); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			files := a.Files()
			if !tc.wantFile {
				if len(files) != 0 {
					t.Fatalf("expected no files, got %d", len(files))
				}
				return
			}

			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}
			if files[0].Filename != "agent-config.yaml" {
				t.Errorf("expected filename %q, got %q", "agent-config.yaml", files[0].Filename)
			}

			var got agenttypes.Config
			if err := yaml.Unmarshal(files[0].Data, &got); err != nil {
				t.Fatalf("failed to unmarshal agent config: %v", err)
			}
			if got.APIVersion != agenttypes.AgentConfigVersion {
				t.Errorf("expected apiVersion %q, got %q", agenttypes.AgentConfigVersion, got.APIVersion)
			}
			if got.Kind != "AgentConfig" {
				t.Errorf("expected kind %q, got %q", "AgentConfig", got.Kind)
			}
			if got.RendezvousIP != tc.wantIP {
				t.Errorf("expected rendezvousIP %q, got %q", tc.wantIP, got.RendezvousIP)
			}
		})
	}
}
