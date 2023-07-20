package recovery

import (
	"fmt"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/coreos"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

// BaseISO generates the base ISO file for the image (CoreOS LiveCD)
type BaseISO struct {
	File *asset.File
}

const (
	coreosIsoName = "coreos-%s.iso"
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

	// Search for disk image in cache dir
	filePattern := fmt.Sprintf(coreosIsoName, applianceConfig.GetCpuArchitecture())
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

	coreOSConfig := coreos.CoreOSConfig{
		EnvConfig:    envConfig,
		ReleaseImage: swag.StringValue(applianceConfig.Config.OcpRelease.URL),
		CpuArch:      applianceConfig.GetCpuArchitecture(),
		PullSecret:   applianceConfig.Config.PullSecret,
	}
	c := coreos.NewCoreOS(coreOSConfig)
	fileName, err := c.DownloadISO()
	if err != nil {
		return log.StopSpinner(spinner, err)
	}

	i.File = &asset.File{Filename: fileName}

	return log.StopSpinner(spinner, nil)
}
