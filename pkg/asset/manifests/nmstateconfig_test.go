package manifests

import (
	"context"
	"strings"
	"testing"

	"github.com/go-openapi/swag/conv"
	"sigs.k8s.io/yaml"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/types"
	aiv1beta1 "github.com/openshift/assisted-service/api/v1beta1"
	"github.com/openshift/installer/pkg/asset"
)

func TestNMStateConfigGenerate(t *testing.T) {
	cases := []struct {
		name          string
		networkConfig []types.NetworkConfigEntry
		wantCount     int
	}{
		{
			name:      "no network config",
			wantCount: 0,
		},
		{
			name: "single network config entry",
			networkConfig: []types.NetworkConfigEntry{
				{
					Interfaces: []types.NetworkInterface{
						{Name: "eth0", MacAddress: "00:11:22:33:44:55"},
					},
					Config: []byte(`interfaces:
  - name: eth0
    type: ethernet
    state: up
    ipv4:
      enabled: true
      address:
        - ip: 192.168.1.10
          prefix-length: 24
`),
				},
			},
			wantCount: 1,
		},
		{
			name: "multiple network config entries",
			networkConfig: []types.NetworkConfigEntry{
				{
					Interfaces: []types.NetworkInterface{
						{Name: "eth0", MacAddress: "00:11:22:33:44:55"},
					},
					Config: []byte(`interfaces: [{name: eth0}]`),
				},
				{
					Interfaces: []types.NetworkInterface{
						{Name: "eth1", MacAddress: "00:11:22:33:44:66"},
					},
					Config: []byte(`interfaces: [{name: eth1}]`),
				},
			},
			wantCount: 2,
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
						NetworkConfig: tc.networkConfig,
					},
				},
			}

			parents := asset.Parents{}
			parents.Add(appCfg)

			nm := &NMStateConfig{}
			if err := nm.Generate(context.Background(), parents); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			files := nm.Files()
			if tc.wantCount == 0 {
				if len(files) != 0 {
					t.Fatalf("expected no files, got %d", len(files))
				}
				return
			}

			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}

			// Split by "---\n" to get individual NMStateConfig objects
			parts := strings.Split(string(files[0].Data), "---\n")
			// Last part is empty after the final "---\n"
			nmStateConfigs := parts[:len(parts)-1]

			if len(nmStateConfigs) != tc.wantCount {
				t.Fatalf("expected %d NMStateConfig objects, got %d", tc.wantCount, len(nmStateConfigs))
			}

			// Validate each NMStateConfig
			for i, data := range nmStateConfigs {
				var got aiv1beta1.NMStateConfig
				if err := yaml.Unmarshal([]byte(data), &got); err != nil {
					t.Fatalf("failed to unmarshal NMStateConfig %d: %v", i, err)
				}

				if got.APIVersion != nmStateConfigAPIVersion {
					t.Errorf("NMStateConfig %d: expected APIVersion %q, got %q", i, nmStateConfigAPIVersion, got.APIVersion)
				}
				if got.Kind != nmStateConfigKind {
					t.Errorf("NMStateConfig %d: expected Kind %q, got %q", i, nmStateConfigKind, got.Kind)
				}

				expectedName := "nmstate-" + string(rune('0'+i))
				if got.Name != expectedName {
					t.Errorf("NMStateConfig %d: expected Name %q, got %q", i, expectedName, got.Name)
				}

				if len(got.Spec.Interfaces) != len(tc.networkConfig[i].Interfaces) {
					t.Errorf("NMStateConfig %d: expected %d interfaces, got %d", i, len(tc.networkConfig[i].Interfaces), len(got.Spec.Interfaces))
				}

				for j, iface := range tc.networkConfig[i].Interfaces {
					if got.Spec.Interfaces[j].Name != iface.Name {
						t.Errorf("NMStateConfig %d interface %d: expected name %q, got %q", i, j, iface.Name, got.Spec.Interfaces[j].Name)
					}
					if got.Spec.Interfaces[j].MacAddress != iface.MacAddress {
						t.Errorf("NMStateConfig %d interface %d: expected MAC %q, got %q", i, j, iface.MacAddress, got.Spec.Interfaces[j].MacAddress)
					}
				}
			}
		})
	}
}
