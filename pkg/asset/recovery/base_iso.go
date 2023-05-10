package recovery

import (
	"fmt"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/coreos"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/release"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

// BaseISO generates the base ISO file for the image (CoreOS LiveCD)
type BaseISO struct {
	File *asset.File
}

const (
	coreosIsoName = "coreos-%s*"
)

var _ asset.Asset = (*BaseISO)(nil)

// Name returns the human-friendly name of the asset.
func (i *BaseISO) Name() string {
	return "Base ISO (CoreOS)"
}

// Dependencies returns dependencies used by the asset.
func (i *BaseISO) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
	}
}

// Generate the base ISO.
func (i *BaseISO) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(envConfig, applianceConfig)

	c := coreos.NewCoreOS(envConfig)
	r := release.NewRelease(*applianceConfig.Config.OcpRelease.URL, applianceConfig.Config.PullSecret, envConfig)
	cpuArch, err := r.GetReleaseArchitecture()
	if err != nil {
		return err
	}
	// Search for disk image in cache dir
	filePattern := fmt.Sprintf(coreosIsoName, cpuArch)
	if fileName := envConfig.FindInCache(filePattern); fileName != "" {
		logrus.Info("Reusing base CoreOS ISO from cache")
		i.File = &asset.File{Filename: fileName}
		return nil
	}

	// Download base CoreOS ISO according to specified release image
	spinner := log.NewSpinner(
		"Downloading CoreOS ISO...",
		"Successfully downloaded CoreOS ISO",
		"Failed to download CoreOS ISO",
		envConfig,
	)
	spinner.FileToMonitor = filePattern
	fileName, err := c.DownloadISO(
		*applianceConfig.Config.OcpRelease.URL,
		applianceConfig.Config.PullSecret)
	if err != nil {
		return log.StopSpinner(spinner, err)
	}

	i.File = &asset.File{Filename: fileName}

	return log.StopSpinner(spinner, nil)
}
