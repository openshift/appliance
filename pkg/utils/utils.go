package utils

import (
	"fmt"
	"strings"

	"github.com/openshift/appliance/pkg/asset/config"
)

// GetOCPVersion returns the OpenShift version string for the InternalReleaseImage.
// If a release URL is provided, it extracts the version tag from the URL.
// Otherwise, it constructs the version from the configured version and architecture.
func GetOCPVersion(applianceConfig *config.ApplianceConfig) string {
	if applianceConfig.Config.OcpRelease.URL != nil && *applianceConfig.Config.OcpRelease.URL != "" {
			return extractVersionFromURL(*applianceConfig.Config.OcpRelease.URL)
	}
	ocpVersion := applianceConfig.Config.OcpRelease.Version
	arch := *applianceConfig.Config.OcpRelease.CpuArchitecture
	return fmt.Sprintf("%s-%s", ocpVersion, arch)
}

// extractVersionFromURL extracts the version tag from a container image URL.
func extractVersionFromURL(url string) string {
	parts := strings.Split(url, ":")
	if len(parts) >= 2 {
			return parts[len(parts)-1]
	}
	return ""
}
