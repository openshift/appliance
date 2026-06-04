package release

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

)

const registriesDDir = "/etc/containers/registries.d"

func isValidRegistryHost(host string) bool {
	return strings.Contains(host, ".") || host == "localhost"
}

func normalizeRegistryHost(entry string) string {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return ""
	}
	if host := imageReferenceRegistryHost(entry); host != "" {
		if isValidRegistryHost(host) {
			return host
		}
		return ""
	}
	host := strings.TrimPrefix(entry, "docker://")
	if strings.Contains(host, "/") {
		return ""
	}
	if colon := strings.LastIndex(host, ":"); colon > 0 && !strings.Contains(host, "]") {
		host = host[:colon]
	}
	if !isValidRegistryHost(host) {
		return ""
	}
	return host
}

func disableSigstoreRegistryHosts(registries *[]string) []string {
	if registries == nil {
		return nil
	}

	registryHostsMap := map[string]struct{}{}
	for _, entry := range *registries {
		if host := normalizeRegistryHost(entry); host != "" {
			registryHostsMap[host] = struct{}{}
		}
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

func disableSigstoreRegistryConfigPath(host string) string {
	return filepath.Join(registriesDDir, host+".yaml")
}

func hostHasRegistriesDConfig(registriesDir, host string, stat func(string) (os.FileInfo, error)) bool {
	_, err := stat(filepath.Join(registriesDir, host+".yaml"))
	return err == nil
}

func filterHostsWithoutExistingRegistriesDConfig(hosts []string, registriesDir string, stat func(string) (os.FileInfo, error)) []string {
	filtered := make([]string, 0, len(hosts))
	for _, host := range hosts {
		if !hostHasRegistriesDConfig(registriesDir, host, stat) {
			filtered = append(filtered, host)
		}
	}
	return filtered
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
	registryHosts := disableSigstoreRegistryHosts(r.ApplianceConfig.Config.DisableSigstoreRegistries)
	if len(registryHosts) == 0 {
		return nil
	}

	registryHosts = filterHostsWithoutExistingRegistriesDConfig(registryHosts, registriesDDir, r.OSInterface.Stat)
	if len(registryHosts) == 0 {
		return nil
	}

	if err := r.OSInterface.MkdirAll(registriesDDir, 0o755); err != nil {
		return err
	}

	for _, host := range registryHosts {
		configPath := disableSigstoreRegistryConfigPath(host)
		if err := r.OSInterface.WriteFile(configPath, buildDisableSigstoreRegistriesConfig([]string{host}), 0o644); err != nil {
			return err
		}
	}

	return nil
}
