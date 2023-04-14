package appliance

import (
	"fmt"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/coreos"
	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/danielerez/openshift-appliance/pkg/release"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

// BaseDiskImage is an asset that generates the base disk image (CoreOS) of OpenShift-based appliance.
type BaseDiskImage struct {
	File *asset.File
}

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

	c := coreos.NewCoreOS(envConfig.CacheDir)
	cpuArch, err := a.getCpuArch(applianceConfig, envConfig.CacheDir)
	if err != nil {
		return err
	}

	// Search for disk image in cache dir
	filePattern := fmt.Sprintf("fedora-coreos*%s.qcow2", cpuArch)
	if fileName := c.FindInCache(filePattern); fileName != "" {
		logrus.Info("Reusing appliance base disk image from cache...")
		a.File = &asset.File{Filename: fileName}
		return nil
	}

	// Download using coreos-installer
	stop := log.Spinner("Downloading appliance base disk image...", "Successfully downloaded appliance base disk image")
	defer stop()
	fileName, err := c.DownloadDiskImage(cpuArch)
	if err != nil {
		return err
	}

	a.File = &asset.File{Filename: fileName}

	return nil
}

// Name returns the human-friendly name of the asset.
func (a *BaseDiskImage) Name() string {
	return "Base disk image (CoreOS)"
}

func (a *BaseDiskImage) getCpuArch(applianceConfig *config.ApplianceConfig, cacheDir string) (string, error) {
	releaseImage := applianceConfig.Config.OcpReleaseImage
	pullSecret := applianceConfig.Config.PullSecret
	r := release.NewRelease(executer.NewExecuter(), releaseImage, pullSecret, cacheDir)
	return r.GetReleaseArchitecture()
}
