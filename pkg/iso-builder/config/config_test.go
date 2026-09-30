package isobuilder

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConfigJSONRoundTrip(t *testing.T) {
	cases := []struct {
		name   string
		config Config
	}{
		{
			name: "full config",
			config: Config{
				OpenshiftVersion:      "4.22",
				ReleaseImageURL:       "quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64",
				PullSecret:            `{"auths":{"cloud.openshift.com":{"auth":"dXNlcjpwYXNz"}}}`,
				SSHKey:                []string{"ssh-rsa AAAA...", "ssh-ed25519 AAAA..."},
				Architecture:          "x86_64",
				Proxy:                 &Proxy{HTTPProxy: "http://proxy:8080", HTTPSProxy: "https://proxy:8443", NoProxy: "localhost,.example.com"},
				AdditionalTrustBundle: "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
				AdditionalNTPServers:  []string{"ntp1.example.com", "ntp2.example.com"},
				FIPS:                  true,
				OLMOperators: []OLMOperator{
					{Name: "kubevirt-hyperconverged", Channel: "stable"},
					{Name: "mtv-operator", Version: "2.12", Channel: "release-v2.12"},
				},
				RendezvousIP: "192.168.1.10",
				NetworkConfig: []json.RawMessage{
					json.RawMessage(`{"interfaces":[{"name":"eth0","type":"ethernet","state":"up"}]}`),
				},
				AdditionalImages: []string{"registry.redhat.io/rhel9/support-tools:latest"},
				ExtraManifests: []ExtraManifest{
					{Name: "custom-mco.yaml", Content: "apiVersion: machineconfiguration.openshift.io/v1\nkind: MachineConfig"},
				},
			},
		},
		{
			name: "minimal config with only pull secret",
			config: Config{
				PullSecret: `{"auths":{}}`,
			},
		},
		{
			name: "version and architecture only",
			config: Config{
				OpenshiftVersion: "4.22",
				PullSecret:       `{"auths":{}}`,
				Architecture:     "aarch64",
			},
		},
		{
			name: "release image URL instead of version",
			config: Config{
				ReleaseImageURL: "quay.io/openshift-release-dev/ocp-release@sha256:abcdef",
				PullSecret:      `{"auths":{}}`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.config)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			var got Config
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if !reflect.DeepEqual(tc.config, got) {
				t.Errorf("round-trip mismatch:\nwant: %+v\ngot:  %+v", tc.config, got)
			}
		})
	}
}

func TestConfigOmitempty(t *testing.T) {
	cfg := Config{PullSecret: `{"auths":{}}`}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if _, ok := raw["pullSecret"]; !ok {
		t.Error("expected pullSecret to be present")
	}

	omittedFields := []string{
		"openshiftVersion", "releaseImageURL", "sshKey", "architecture",
		"proxy", "additionalTrustBundle", "additionalNTPServers", "fips",
		"olmOperators", "rendezvousIP", "networkConfig", "additionalImages",
		"extraManifests",
	}
	for _, field := range omittedFields {
		if _, ok := raw[field]; ok {
			t.Errorf("expected %s to be omitted when empty", field)
		}
	}
}

func TestConfigNetworkConfigPreservation(t *testing.T) {
	nmstate := `{"interfaces":[{"name":"ens3","type":"ethernet","state":"up","ipv4":{"enabled":true,"dhcp":true}}],"dns-resolver":{"config":{"server":["8.8.8.8"]}}}`

	cfg := Config{
		PullSecret:    `{"auths":{}}`,
		NetworkConfig: []json.RawMessage{json.RawMessage(nmstate)},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(got.NetworkConfig) != 1 {
		t.Fatalf("expected 1 network config entry, got %d", len(got.NetworkConfig))
	}

	if string(got.NetworkConfig[0]) != nmstate {
		t.Errorf("network config not preserved:\nwant: %s\ngot:  %s", nmstate, string(got.NetworkConfig[0]))
	}
}

func TestConfigNilSlicesOmitted(t *testing.T) {
	cfg := Config{PullSecret: `{"auths":{}}`, SSHKey: nil}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if _, ok := raw["sshKey"]; ok {
		t.Error("expected nil sshKey to be omitted from JSON")
	}
}

func TestConfigPopulatedSlicePresent(t *testing.T) {
	cfg := Config{PullSecret: `{"auths":{}}`, SSHKey: []string{"ssh-rsa AAAA..."}}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if _, ok := raw["sshKey"]; !ok {
		t.Error("expected populated sshKey to be present in JSON")
	}
}
