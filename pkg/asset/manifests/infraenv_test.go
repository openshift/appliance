package manifests

import (
	"context"
	"testing"

	"github.com/go-openapi/swag/conv"
	"sigs.k8s.io/yaml"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/types"
	aiv1beta1 "github.com/openshift/assisted-service/api/v1beta1"
	"github.com/openshift/installer/pkg/asset"
)

func TestInfraEnvGenerateSSHKey(t *testing.T) {
	cases := []struct {
		name       string
		sshKey     *string
		wantSSHKey string
	}{
		{
			name:       "nil ssh key is not set",
			wantSSHKey: "",
		},
		{
			name:       "ssh key is set in infraenv",
			sshKey:     conv.Pointer("ssh-rsa AAAA... user@host"),
			wantSSHKey: "ssh-rsa AAAA... user@host",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appCfg := &config.ApplianceConfig{
				Config: &types.ApplianceConfig{
					OcpRelease: types.ReleaseImage{
						CpuArchitecture: conv.Pointer("x86_64"),
					},
					SshKey: tc.sshKey,
				},
			}

			parents := asset.Parents{}
			parents.Add(appCfg)

			ie := &InfraEnv{}
			if err := ie.Generate(context.Background(), parents); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			files := ie.Files()
			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}

			var got aiv1beta1.InfraEnv
			if err := yaml.Unmarshal(files[0].Data, &got); err != nil {
				t.Fatalf("failed to unmarshal infraenv: %v", err)
			}
			if got.Spec.SSHAuthorizedKey != tc.wantSSHKey {
				t.Errorf("expected SSHAuthorizedKey %q, got %q", tc.wantSSHKey, got.Spec.SSHAuthorizedKey)
			}
		})
	}
}
