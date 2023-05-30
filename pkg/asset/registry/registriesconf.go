package registry

import (
	"fmt"

	"github.com/containers/image/pkg/sysregistriesv2"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/installer/pkg/asset"
	"github.com/pelletier/go-toml"
)

const (
	RegistryDomain = "registry.appliance.com"
)

// RegistriesConf generates the registries.conf file.
type RegistriesConf struct {
	FileData []byte
}

var _ asset.Asset = (*RegistriesConf)(nil)

// Name returns a human friendly name for the asset.
func (*RegistriesConf) Name() string {
	return "Mirror Registries Config"
}

// Dependencies returns all the dependencies directly needed to generate
// the asset.
func (*RegistriesConf) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
	}
}

// Generate generates the registries.conf file from install-config.
func (i *RegistriesConf) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	dependencies.Get(envConfig)

	registries := &sysregistriesv2.V2RegistriesConf{
		Registries: []sysregistriesv2.Registry{
			{
				Endpoint: sysregistriesv2.Endpoint{
					Location: "quay.io/openshift-release-dev/ocp-release",
				},
				Mirrors: []sysregistriesv2.Endpoint{
					{
						Location: fmt.Sprintf("%s:5000/openshift/release-images", RegistryDomain),
					},
				},
			},
			{
				Endpoint: sysregistriesv2.Endpoint{
					Location: "quay.io/openshift-release-dev/ocp-v4.0-art-dev",
				},
				Mirrors: []sysregistriesv2.Endpoint{
					{
						Location: fmt.Sprintf("%s:5000/openshift/release", RegistryDomain),
					},
				},
			},
			// TODO: remove once not using custom AGENT_DOCKER_IMAGE from quay.io
			{
				Endpoint: sysregistriesv2.Endpoint{
					Location: "quay.io",
				},
				Mirrors: []sysregistriesv2.Endpoint{
					{
						Location: fmt.Sprintf("%s:5000", RegistryDomain),
					},
				},
			},
		},
	}

	registriesData, err := toml.Marshal(registries)
	if err != nil {
		return err
	}

	i.FileData = registriesData

	return nil
}
