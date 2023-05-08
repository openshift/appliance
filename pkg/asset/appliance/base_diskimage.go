package appliance

import (
	"fmt"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/coreos"
	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/danielerez/openshift-appliance/pkg/release"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

// BaseDiskImage is an asset that generates the base disk image (CoreOS) of OpenShift-based appliance.
type BaseDiskImage struct {
	File *asset.File
}

const (
	coreosImageName = "rhcos-*%s.qcow2"
)

var _ asset.Asset = (*BaseDiskImage)(nil)

// Dependencies returns the assets on which the Bootstrap asset depends.
func (a *BaseDiskImage) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
	}
}

// Generate the base disk image.
func (a *BaseDiskImage) Generate(dependencies asset.Parents) error {
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
	filePattern := fmt.Sprintf(coreosImageName, cpuArch)
	if fileName := envConfig.FindInCache(filePattern); fileName != "" {
		logrus.Info("Reusing appliance base disk image from cache")
		a.File = &asset.File{Filename: fileName}
		return nil
	}

	// Download using coreos-installer
	spinner := log.NewSpinner(
		"Downloading appliance base disk image...",
		"Successfully downloaded appliance base disk image",
		"Failed to download appliance base disk image",
	)
	fileName, err := c.DownloadDiskImage(*applianceConfig.Config.OcpRelease.URL, applianceConfig.Config.PullSecret)
	if err != nil {
		return log.StopSpinner(spinner, err)
	}

	a.File = &asset.File{Filename: fileName}

	return log.StopSpinner(spinner, nil)
}

// Name returns the human-friendly name of the asset.
func (a *BaseDiskImage) Name() string {
	return "Base disk image (CoreOS)"
}
