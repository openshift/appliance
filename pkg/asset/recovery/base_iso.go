package recovery

import (
	"fmt"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/coreos"
	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/danielerez/openshift-appliance/pkg/release"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

// BaseISO generates the base ISO file for the image (CoreOS LiveCD)
type BaseISO struct {
	File *asset.File
}

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
	r := release.NewRelease(*applianceConfig.Config.OcpReleaseImage, applianceConfig.Config.PullSecret, envConfig)
	cpuArch, err := r.GetReleaseArchitecture()
	if err != nil {
		return err
	}
	// Search for disk image in cache dir
	filePattern := fmt.Sprintf("coreos-%s*", cpuArch)
	if fileName := c.FindInCache(filePattern); fileName != "" {
		logrus.Info("Reusing base ISO from cache...")
		i.File = &asset.File{Filename: fileName}
		return nil
	}

	// Download base CoreOS ISO according to specified release image
	stop := log.Spinner("Downloading CoreOS ISO...", "Successfully downloaded CoreOS ISO")
	defer stop()
	fileName, err := c.DownloadISO(
		*applianceConfig.Config.OcpReleaseImage,
		applianceConfig.Config.PullSecret)
	if err != nil {
		return err
	}

	i.File = &asset.File{Filename: fileName}

	return nil
}
