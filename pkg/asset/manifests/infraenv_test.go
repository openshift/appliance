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

func TestInfraEnvGenerateAdditionalTrustBundle(t *testing.T) {
	cases := []struct {
		name            string
		trustBundle     string
		wantTrustBundle string
	}{
		{
			name:            "no trust bundle",
			wantTrustBundle: "",
		},
		{
			name:            "trust bundle is set in infraenv",
			trustBundle:     "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
			wantTrustBundle: "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
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
						AdditionalTrustBundle: tc.trustBundle,
					},
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
			if got.Spec.AdditionalTrustBundle != tc.wantTrustBundle {
				t.Errorf("expected AdditionalTrustBundle %q, got %q", tc.wantTrustBundle, got.Spec.AdditionalTrustBundle)
			}
		})
	}
}

func TestInfraEnvGenerateAdditionalNTPSources(t *testing.T) {
	cases := []struct {
		name       string
		ntpSources []string
		wantNTP    []string
	}{
		{
			name:    "no NTP sources",
			wantNTP: nil,
		},
		{
			name:       "NTP sources are set in infraenv",
			ntpSources: []string{"0.rhel.pool.ntp.org", "1.rhel.pool.ntp.org"},
			wantNTP:    []string{"0.rhel.pool.ntp.org", "1.rhel.pool.ntp.org"},
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
						AdditionalNTPSources: tc.ntpSources,
					},
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

			if tc.wantNTP == nil {
				if len(got.Spec.AdditionalNTPSources) != 0 {
					t.Errorf("expected no NTP sources, got %v", got.Spec.AdditionalNTPSources)
				}
				return
			}

			if len(got.Spec.AdditionalNTPSources) != len(tc.wantNTP) {
				t.Fatalf("expected %d NTP sources, got %d", len(tc.wantNTP), len(got.Spec.AdditionalNTPSources))
			}
			for i, want := range tc.wantNTP {
				if got.Spec.AdditionalNTPSources[i] != want {
					t.Errorf("NTP[%d]: expected %q, got %q", i, want, got.Spec.AdditionalNTPSources[i])
				}
			}
		})
	}
}
