package commands

import (
	"testing"

	"github.com/go-openapi/swag/conv"

	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
	"github.com/openshift/appliance/pkg/types"
)

func TestConvertToApplianceConfigAdditionalImages(t *testing.T) {
	cases := []struct {
		name           string
		embeddedImages []string
		cliImages      []string
		wantImages     []types.Image
		wantNil        bool
	}{
		{
			name:    "no additional images",
			wantNil: true,
		},
		{
			name:           "only embedded images",
			embeddedImages: []string{"registry.example.com/app:v1"},
			wantImages:     []types.Image{{Name: "registry.example.com/app:v1"}},
		},
		{
			name:       "only CLI images",
			cliImages:  []string{"registry.example.com/tool:latest"},
			wantImages: []types.Image{{Name: "registry.example.com/tool:latest"}},
		},
		{
			name:           "embedded and CLI images merged",
			embeddedImages: []string{"registry.example.com/embedded:v1"},
			cliImages:      []string{"registry.example.com/cli1:v2", "registry.example.com/cli2:v3"},
			wantImages: []types.Image{
				{Name: "registry.example.com/embedded:v1"},
				{Name: "registry.example.com/cli1:v2"},
				{Name: "registry.example.com/cli2:v3"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &isobuilderconfig.Config{
				OpenshiftVersion: "4.22",
				PullSecret:       `{"auths":{}}`,
				AdditionalImages: tc.embeddedImages,
			}
			cfg.AdditionalImages = append(cfg.AdditionalImages, tc.cliImages...)

			b := &Builder{workingDir: "."}
			appCfg := b.convertToApplianceConfig(cfg)

			if tc.wantNil {
				if appCfg.AdditionalImages != nil {
					t.Fatalf("expected nil AdditionalImages, got %v", *appCfg.AdditionalImages)
				}
				return
			}

			if appCfg.AdditionalImages == nil {
				t.Fatal("expected non-nil AdditionalImages, got nil")
			}

			got := *appCfg.AdditionalImages
			if len(got) != len(tc.wantImages) {
				t.Fatalf("expected %d images, got %d: %v", len(tc.wantImages), len(got), got)
			}
			for i, want := range tc.wantImages {
				if got[i].Name != want.Name {
					t.Errorf("image[%d]: expected %q, got %q", i, want.Name, got[i].Name)
				}
			}
		})
	}
}

func TestConvertToApplianceConfigRendezvousIP(t *testing.T) {
	cases := []struct {
		name         string
		rendezvousIP string
		wantIP       *string
	}{
		{
			name:   "empty rendezvous IP is not set",
			wantIP: nil,
		},
		{
			name:         "valid IPv4 address",
			rendezvousIP: "192.168.1.10",
			wantIP:       conv.Pointer("192.168.1.10"),
		},
		{
			name:         "valid IPv6 address",
			rendezvousIP: "fd00::1",
			wantIP:       conv.Pointer("fd00::1"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &isobuilderconfig.Config{
				OpenshiftVersion: "4.22",
				PullSecret:       `{"auths":{}}`,
				RendezvousIP:     tc.rendezvousIP,
			}

			b := &Builder{workingDir: "."}
			appCfg := b.convertToApplianceConfig(cfg)

			if tc.wantIP == nil {
				if appCfg.RendezvousIP != nil {
					t.Fatalf("expected nil RendezvousIP, got %q", *appCfg.RendezvousIP)
				}
				return
			}

			if appCfg.RendezvousIP == nil {
				t.Fatal("expected non-nil RendezvousIP, got nil")
			}
			if *appCfg.RendezvousIP != *tc.wantIP {
				t.Errorf("expected RendezvousIP %q, got %q", *tc.wantIP, *appCfg.RendezvousIP)
			}
		})
	}
}

func TestConvertToApplianceConfigAdditionalNTPSources(t *testing.T) {
	cases := []struct {
		name       string
		ntpServers []string
		wantNTP    []string
	}{
		{
			name:    "no NTP servers",
			wantNTP: nil,
		},
		{
			name:       "single NTP server",
			ntpServers: []string{"0.rhel.pool.ntp.org"},
			wantNTP:    []string{"0.rhel.pool.ntp.org"},
		},
		{
			name:       "multiple NTP servers",
			ntpServers: []string{"0.rhel.pool.ntp.org", "1.rhel.pool.ntp.org"},
			wantNTP:    []string{"0.rhel.pool.ntp.org", "1.rhel.pool.ntp.org"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &isobuilderconfig.Config{
				OpenshiftVersion:     "4.22",
				PullSecret:           `{"auths":{}}`,
				AdditionalNTPServers: tc.ntpServers,
			}

			b := &Builder{workingDir: "."}
			appCfg := b.convertToApplianceConfig(cfg)

			if tc.wantNTP == nil {
				if len(appCfg.AdditionalNTPSources) != 0 {
					t.Fatalf("expected no NTP sources, got %v", appCfg.AdditionalNTPSources)
				}
				return
			}

			if len(appCfg.AdditionalNTPSources) != len(tc.wantNTP) {
				t.Fatalf("expected %d NTP sources, got %d", len(tc.wantNTP), len(appCfg.AdditionalNTPSources))
			}
			for i, want := range tc.wantNTP {
				if appCfg.AdditionalNTPSources[i] != want {
					t.Errorf("NTP[%d]: expected %q, got %q", i, want, appCfg.AdditionalNTPSources[i])
				}
			}
		})
	}
}
