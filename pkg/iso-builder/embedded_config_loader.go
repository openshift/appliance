package isobuilder

import (
	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
)

// loadEmbeddedConfig decodes the configuration from the binary's embedded area.
// Internal utility for direct access to rawConfigArea — used by both Builder
// and ShowConfig.
func loadEmbeddedConfig() (*isobuilderconfig.Config, error) {
	return isobuilderconfig.ReadFromData([]byte(rawConfigArea))
}
