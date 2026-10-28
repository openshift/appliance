package manifests

import (
	"context"
	"testing"

	"github.com/go-openapi/swag/conv"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/types"
	"github.com/openshift/installer/pkg/asset"
)

func TestAgentPullSecretGenerate(t *testing.T) {
	cases := []struct {
		name           string
		agentPullSecret string
		wantPullSecret string
	}{
		{
			name:           "no agent pull secret uses dummy",
			wantPullSecret: dummyPullSecret,
		},
		{
			name:            "agent pull secret overrides dummy",
			agentPullSecret: `{"auths":{"registry.example.com":{"auth":"dGVzdDp0ZXN0"}}}`,
			wantPullSecret:  `{"auths":{"registry.example.com":{"auth":"dGVzdDp0ZXN0"}}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appCfg := &config.ApplianceConfig{
				Config: &types.ApplianceConfig{
					OcpRelease: types.ReleaseImage{
						CpuArchitecture: conv.Pointer("x86_64"),
					},
					ApplianceConfigExt: types.ApplianceConfigExt{
						AgentPullSecret: tc.agentPullSecret,
					},
				},
			}

			parents := asset.Parents{}
			parents.Add(appCfg)

			aps := &AgentPullSecret{}
			if err := aps.Generate(context.Background(), parents); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			files := aps.Files()
			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}

			var got corev1.Secret
			if err := yaml.Unmarshal(files[0].Data, &got); err != nil {
				t.Fatalf("failed to unmarshal secret: %v", err)
			}

			if got.StringData[pullSecretKey] != tc.wantPullSecret {
				t.Errorf("expected pull secret %q, got %q", tc.wantPullSecret, got.StringData[pullSecretKey])
			}
		})
	}
}
