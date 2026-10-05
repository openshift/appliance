package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
)

func TestFormatConfig(t *testing.T) {
	const secretValue = `{"auths":{"cloud.openshift.com":{"auth":"dXNlcjpwYXNz"}}}`

	cases := []struct {
		name     string
		config   *isobuilderconfig.Config
		contains []string
		absent   []string
	}{
		{
			name: "full config",
			config: &isobuilderconfig.Config{
				OpenshiftVersion:      "4.22",
				ReleaseImageURL:       "quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64",
				PullSecret:            secretValue,
				SSHKey:                []string{"ssh-rsa AAAA...", "ssh-ed25519 AAAA..."},
				Architecture:          "x86_64",
				Proxy:                 &isobuilderconfig.Proxy{HTTPProxy: "http://proxy:8080", HTTPSProxy: "https://proxy:8443", NoProxy: "localhost,.example.com"},
				AdditionalTrustBundle: "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
				AdditionalNTPServers:  []string{"ntp1.example.com", "ntp2.example.com"},
				FIPS:                  true,
				OLMOperators: []isobuilderconfig.OLMOperator{
					{Name: "kubevirt-hyperconverged", Channel: "stable"},
					{Name: "mtv-operator", Version: "2.12", Channel: "release-v2.12"},
				},
				RendezvousIP: "192.168.1.10",
				NetworkConfig: []json.RawMessage{
					json.RawMessage(`{"interfaces":[{"name":"eth0"}]}`),
				},
				AdditionalImages: []string{"registry.redhat.io/rhel9/support-tools:latest"},
				ExtraManifests: []isobuilderconfig.ExtraManifest{
					{Name: "custom-mco.yaml", Content: "apiVersion: v1"},
				},
			},
			contains: []string{
				"OpenShift Version:",
				"4.22",
				"Release Image:",
				"quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64",
				"Architecture:",
				"x86_64",
				"Pull Secret:",
				"***",
				"SSH Keys:",
				"2 key(s)",
				"FIPS:",
				"Enabled",
				"Rendezvous IP:",
				"192.168.1.10",
				"Proxy:",
				"HTTP Proxy:",
				"http://proxy:8080",
				"HTTPS Proxy:",
				"https://proxy:8443",
				"No Proxy:",
				"localhost,.example.com",
				"Additional Trust Bundle:",
				"present",
				"NTP Servers:",
				"ntp1.example.com",
				"ntp2.example.com",
				"OLM Operators:",
				"kubevirt-hyperconverged (channel: stable)",
				"mtv-operator v2.12 (channel: release-v2.12)",
				"Additional Images:",
				"registry.redhat.io/rhel9/support-tools:latest",
				"Network Configs:",
				"1 config(s)",
				"Extra Manifests:",
				"custom-mco.yaml",
			},
			absent: []string{
				secretValue,
				"BEGIN CERTIFICATE",
				"apiVersion: v1",
			},
		},
		{
			name: "minimal config with only pull secret",
			config: &isobuilderconfig.Config{
				PullSecret: secretValue,
			},
			contains: []string{
				"Pull Secret:",
				"***",
			},
			absent: []string{
				secretValue,
				"OpenShift Version:",
				"Architecture:",
				"Release Image:",
				"SSH Keys:",
				"FIPS:",
				"Rendezvous IP:",
				"Proxy:",
				"NTP Servers:",
				"OLM Operators:",
				"Additional Images:",
				"Network Configs:",
				"Extra Manifests:",
				"Additional Trust Bundle:",
			},
		},
		{
			name: "FIPS enabled",
			config: &isobuilderconfig.Config{
				PullSecret: secretValue,
				FIPS:       true,
			},
			contains: []string{"FIPS:", "Enabled"},
		},
		{
			name: "FIPS disabled",
			config: &isobuilderconfig.Config{
				PullSecret: secretValue,
				FIPS:       false,
			},
			absent: []string{"FIPS:"},
		},
		{
			name: "proxy fields",
			config: &isobuilderconfig.Config{
				PullSecret: secretValue,
				Proxy: &isobuilderconfig.Proxy{
					HTTPProxy:  "http://proxy:3128",
					HTTPSProxy: "https://proxy:3129",
					NoProxy:    "10.0.0.0/8",
				},
			},
			contains: []string{
				"Proxy:",
				"HTTP Proxy:",
				"http://proxy:3128",
				"HTTPS Proxy:",
				"https://proxy:3129",
				"No Proxy:",
				"10.0.0.0/8",
			},
		},
		{
			name: "trust bundle summary",
			config: &isobuilderconfig.Config{
				PullSecret:            secretValue,
				AdditionalTrustBundle: "-----BEGIN CERTIFICATE-----\nABCDEF\n-----END CERTIFICATE-----",
			},
			contains: []string{
				"Additional Trust Bundle:",
				"present",
			},
			absent: []string{
				"BEGIN CERTIFICATE",
				"ABCDEF",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := formatConfig(&buf, tc.config); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			output := buf.String()

			for _, s := range tc.contains {
				if !strings.Contains(output, s) {
					t.Errorf("expected output to contain %q, got:\n%s", s, output)
				}
			}
			for _, s := range tc.absent {
				if strings.Contains(output, s) {
					t.Errorf("expected output to NOT contain %q, got:\n%s", s, output)
				}
			}
		})
	}
}

// TestFormatConfigOutput documents the exact output of show-config.
// This test serves as a reference for the expected human-readable format.
func TestFormatConfigOutput(t *testing.T) {
	cfg := &isobuilderconfig.Config{
		OpenshiftVersion:      "4.22",
		ReleaseImageURL:       "quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64",
		PullSecret:            `{"auths":{"cloud.openshift.com":{"auth":"dXNlcjpwYXNz"}}}`,
		SSHKey:                []string{"ssh-rsa AAAA...", "ssh-ed25519 AAAA..."},
		Architecture:          "x86_64",
		Proxy:                 &isobuilderconfig.Proxy{HTTPProxy: "http://proxy:8080", HTTPSProxy: "https://proxy:8443", NoProxy: "localhost,.example.com"},
		AdditionalTrustBundle: "-----BEGIN CERTIFICATE-----\nMIIDxTCCAq2gAwIBAgIJA...\n-----END CERTIFICATE-----",
		AdditionalNTPServers:  []string{"ntp1.example.com", "ntp2.example.com"},
		FIPS:                  true,
		OLMOperators: []isobuilderconfig.OLMOperator{
			{Name: "kubevirt-hyperconverged", Channel: "stable"},
			{Name: "mtv-operator", Version: "2.12", Channel: "release-v2.12"},
		},
		RendezvousIP: "192.168.1.10",
		NetworkConfig: []json.RawMessage{
			json.RawMessage(`{"interfaces":[{"name":"eth0","type":"ethernet","state":"up"}]}`),
			json.RawMessage(`{"interfaces":[{"name":"eth1","type":"ethernet","state":"up"}]}`),
		},
		AdditionalImages: []string{
			"registry.redhat.io/rhel9/support-tools:latest",
			"registry.redhat.io/ubi9/ubi:latest",
		},
		ExtraManifests: []isobuilderconfig.ExtraManifest{
			{Name: "custom-mco.yaml", Content: "apiVersion: machineconfiguration.openshift.io/v1\nkind: MachineConfig"},
			{Name: "extra-configmap.yaml", Content: "apiVersion: v1\nkind: ConfigMap"},
		},
	}

	expected := `OpenShift Version:        4.22
Release Image:            quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64
Architecture:             x86_64
Pull Secret:              ***
SSH Keys:                 2 key(s)
FIPS:                     Enabled
Rendezvous IP:            192.168.1.10

Proxy:
  HTTP Proxy:             http://proxy:8080
  HTTPS Proxy:            https://proxy:8443
  No Proxy:               localhost,.example.com
Additional Trust Bundle:  present (78 bytes)
NTP Servers:              ntp1.example.com, ntp2.example.com

OLM Operators:
  - kubevirt-hyperconverged (channel: stable)
  - mtv-operator v2.12 (channel: release-v2.12)

Additional Images:
  - registry.redhat.io/rhel9/support-tools:latest
  - registry.redhat.io/ubi9/ubi:latest
Network Configs:          2 config(s)

Extra Manifests:
  - custom-mco.yaml
  - extra-configmap.yaml
`

	var buf bytes.Buffer
	if err := formatConfig(&buf, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != expected {
		t.Errorf("output mismatch.\nExpected:\n%s\nGot:\n%s", expected, buf.String())
	}
}

func TestShowConfigMissingConfig(t *testing.T) {
	var buf bytes.Buffer
	err := ShowConfig(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No configuration has been found") {
		t.Errorf("unexpected output: %s", buf.String())
	}
}
