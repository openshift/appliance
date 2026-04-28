package release

import (
	"sort"
	"strings"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/types"
)

const (
	disableSigstoreFilePath = "/etc/containers/registries.d/00-appliance-disable-sigstore.yaml"
	registriesDDir          = "/etc/containers/registries.d"
)

func additionalImageRegistryHosts(images *[]types.Image) []string {
	if images == nil {
		return nil
	}

	registryHostsMap := map[string]struct{}{}
	for _, img := range *images {
		registryHost := imageReferenceRegistryHost(img.Name)
		if registryHost == "" {
			continue
		}
		registryHostsMap[registryHost] = struct{}{}
	}

	registryHosts := make([]string, 0, len(registryHostsMap))
	for registryHost := range registryHostsMap {
		registryHosts = append(registryHosts, registryHost)
	}
	sort.Strings(registryHosts)

	return registryHosts
}

func imageReferenceRegistryHost(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), "docker://")
	if name == "" {
		return ""
	}
	if idx := strings.Index(name, "/"); idx > 0 {
		host := name[:idx]
		// host:port — strip port; ignore IPv6 in brackets (unusual for this use case)
		if colon := strings.LastIndex(host, ":"); colon > 0 && !strings.Contains(host, "]") {
			host = host[:colon]
		}
		return host
	}
	return ""
}

// buildDisableSigstoreRegistriesConfig creates a containers-registries.d snippet
// that disables sigstore attachments for the given registry hosts.
// This is used as a workaround for oc-mirror v2 flows where some registries do not
// publish OCI .sig attachment manifests and mirroring fails when they are queried.
func buildDisableSigstoreRegistriesConfig(registryHosts []string) []byte {
	var builder strings.Builder

	builder.WriteString("docker:\n")

	for _, registryHost := range registryHosts {
		builder.WriteString("  ")
		builder.WriteString(registryHost)
		builder.WriteString(":\n")
		builder.WriteString("    use-sigstore-attachments: false\n")
	}

	return []byte(builder.String())
}

func (r *release) disableSigstoreForRelevantRegistries() error {
	if !swag.BoolValue(r.ApplianceConfig.Config.DisableSigstoreForAdditionalImages) {
		return nil
	}

	registryHosts := additionalImageRegistryHosts(r.ApplianceConfig.Config.AdditionalImages)
	if len(registryHosts) == 0 {
		return nil
	}

	if err := r.OSInterface.MkdirAll(registriesDDir, 0o755); err != nil {
		return err
	}

	return r.OSInterface.WriteFile(disableSigstoreFilePath, buildDisableSigstoreRegistriesConfig(registryHosts), 0o644)
}
