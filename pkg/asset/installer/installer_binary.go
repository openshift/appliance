package installer

import (

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/installer"
	"github.com/openshift/installer/pkg/asset"
	"github.com/pkg/errors"
)

type InstallerBinary struct {
	URL string
}

var _ asset.Asset = (*InstallerBinary)(nil)

func (a *InstallerBinary) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
	}
}

func (a *InstallerBinary) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(envConfig, applianceConfig)

	installerConfig := installer.InstallerConfig{
		EnvConfig:       envConfig,
		ApplianceConfig: applianceConfig,
	}
	inst := installer.NewInstaller(installerConfig)
	installerDownloadURL, err := inst.GetInstallerDownloadURL()
	if err != nil {
		return errors.Wrapf(err, "Failed to generate installer download URL")
	}
	a.URL = installerDownloadURL

	return nil
}

// Name returns the human-friendly name of the asset.
func (a *InstallerBinary) Name() string {
	return "Installer Binary"
}
