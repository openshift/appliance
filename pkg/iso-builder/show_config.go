package isobuilder

import (
	"fmt"
	"io"
	"strings"

	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
)

const redactedValue = "***"

// ShowConfig loads the embedded configuration and writes a human-readable,
// redacted summary to w.
func ShowConfig(w io.Writer) error {
	cfg, err := loadEmbeddedConfig()
	if err != nil {
		if strings.Contains(err.Error(), "no embedded configuration") {
			_, err = fmt.Fprintln(w, "No configuration has been found in this binary.")
			return err
		}
		return err
	}
	return formatConfig(w, cfg)
}

func formatConfig(w io.Writer, cfg *isobuilderconfig.Config) error {
	p := &printer{w: w}

	p.field("OpenShift Version", cfg.OpenshiftVersion)
	p.field("Release Image", cfg.ReleaseImageURL)
	p.field("Architecture", cfg.Architecture)
	p.printf("%-25s %s\n", "Pull Secret:", redactedValue)

	if len(cfg.SSHKey) > 0 {
		p.printf("%-25s %d key(s)\n", "SSH Keys:", len(cfg.SSHKey))
	}

	if cfg.FIPS {
		p.printf("%-25s %s\n", "FIPS:", "Enabled")
	}

	p.field("Rendezvous IP", cfg.RendezvousIP)

	if cfg.Proxy != nil {
		p.printf("\nProxy:\n")
		p.field("  HTTP Proxy", cfg.Proxy.HTTPProxy)
		p.field("  HTTPS Proxy", cfg.Proxy.HTTPSProxy)
		p.field("  No Proxy", cfg.Proxy.NoProxy)
	}

	if cfg.AdditionalTrustBundle != "" {
		p.printf("%-25s present (%d bytes)\n", "Additional Trust Bundle:", len(cfg.AdditionalTrustBundle))
	}

	if len(cfg.AdditionalNTPServers) > 0 {
		p.printf("%-25s %s\n", "NTP Servers:", strings.Join(cfg.AdditionalNTPServers, ", "))
	}

	if len(cfg.OLMOperators) > 0 {
		p.printf("\nOLM Operators:\n")
		for _, op := range cfg.OLMOperators {
			p.printf("  - %s\n", formatOperator(op))
		}
	}

	if len(cfg.AdditionalImages) > 0 {
		p.printf("\nAdditional Images:\n")
		for _, img := range cfg.AdditionalImages {
			p.printf("  - %s\n", img)
		}
	}

	if len(cfg.NetworkConfig) > 0 {
		p.printf("%-25s %d config(s)\n", "Network Configs:", len(cfg.NetworkConfig))
	}

	if len(cfg.ExtraManifests) > 0 {
		p.printf("\nExtra Manifests:\n")
		for _, m := range cfg.ExtraManifests {
			p.printf("  - %s\n", m.Name)
		}
	}

	return p.err
}

type printer struct {
	w   io.Writer
	err error
}

func (p *printer) printf(format string, a ...any) {
	if p.err == nil {
		_, p.err = fmt.Fprintf(p.w, format, a...)
	}
}

func (p *printer) field(label, value string) {
	if value != "" {
		p.printf("%-25s %s\n", label+":", value)
	}
}

func formatOperator(op isobuilderconfig.OLMOperator) string {
	s := op.Name
	if op.Version != "" {
		s += " v" + op.Version
	}
	if op.Channel != "" {
		s += " (channel: " + op.Channel + ")"
	}
	return s
}
