package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/image/pkg/sysregistriesv2"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/installer/pkg/asset"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

const (
	RegistryDomain      = "registry.appliance.com"
	RegistryPort        = 5000
	RegistryPortUpgrade = 5001
)

var (
	registriesConfFilename = filepath.Join("mirror", "registries.conf")
)

// RegistriesConf generates the registries.conf file.
type RegistriesConf struct {
	File   *asset.File
	Config *sysregistriesv2.V2RegistriesConf
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
func (i *RegistriesConf) Generate(_ context.Context, dependencies asset.Parents) error {
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
						Location: fmt.Sprintf("%s:%d/openshift/release-images", RegistryDomain, RegistryPort),
					},
					// Mirror for the upgrade registry
					{
						Location: fmt.Sprintf("%s:%d/openshift/release-images", RegistryDomain, RegistryPortUpgrade),
					},
				},
			},
			{
				Endpoint: sysregistriesv2.Endpoint{
					Location: "quay.io/openshift-release-dev/ocp-v4.0-art-dev",
				},
				Mirrors: []sysregistriesv2.Endpoint{
					{
						Location: fmt.Sprintf("%s:%d/openshift/release", RegistryDomain, RegistryPort),
					},
					// Mirror for the upgrade registry
					{
						Location: fmt.Sprintf("%s:%d/openshift/release", RegistryDomain, RegistryPortUpgrade),
					},
				},
			},
		},
	}

	registriesData, err := toml.Marshal(registries)
	if err != nil {
		return err
	}

	i.File = &asset.File{
		Filename: registriesConfFilename,
		Data:     registriesData,
	}

	return nil
}

func (i *RegistriesConf) Load(f asset.FileFetcher) (bool, error) {
	file, err := f.FetchByName(registriesConfFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, fmt.Sprintf("failed to load %s file", registriesConfFilename))
	}

	registriesConf := &sysregistriesv2.V2RegistriesConf{}
	if err := toml.Unmarshal(file.Data, registriesConf); err != nil {
		return false, errors.Wrapf(err, "failed to unmarshal %s", registriesConfFilename)
	}

	i.File, i.Config = file, registriesConf

	return true, nil
}

// Files returns the files generated by the asset.
func (i *RegistriesConf) Files() []*asset.File {
	if i.File != nil {
		return []*asset.File{i.File}
	}
	return []*asset.File{}
}
