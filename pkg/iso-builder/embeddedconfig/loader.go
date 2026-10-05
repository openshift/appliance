package embeddedconfig

import (
	_ "embed"

	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
)

//go:generate go run ../gen_embed_area

//go:embed config_embed_area.bin
var rawConfigArea string

// LoadConfig decodes the configuration from the binary's embedded area.
func LoadConfig() (*isobuilderconfig.Config, error) {
	return isobuilderconfig.ReadFromData([]byte(rawConfigArea))
}
