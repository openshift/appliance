package recovery

import (
	"fmt"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/coreos"
	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/danielerez/openshift-appliance/pkg/release"

	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/asset"
)

// BaseIso generates the base ISO file for the image (CoreOS LiveCD)
type BaseISO struct {
	File           *asset.File
	LiveISOVersion string
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
	logrus.Info("Downloading CoreOS ISO...")

	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(envConfig, applianceConfig)

	c := coreos.NewCoreOS(envConfig.CacheDir)
	cpuArch, err := i.getCpuArch(applianceConfig, envConfig.CacheDir)
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
	fileName, err := c.DownloadISO(
		applianceConfig.Config.OcpReleaseImage,
		applianceConfig.Config.PullSecret)
	if err != nil {
		return err
	}

	// Get and store LiveISO version of the ISO
	liveISO, err := c.GetLiveISOVersion(fileName)
	if err != nil {
		return err
	}
	i.LiveISOVersion = liveISO
	i.File = &asset.File{Filename: fileName}

	return nil
}

func (i *BaseISO) getCpuArch(applianceConfig *config.ApplianceConfig, cacheDir string) (string, error) {
	releaseImage := applianceConfig.Config.OcpReleaseImage
	pullSecret := applianceConfig.Config.PullSecret
	r := release.NewRelease(executer.NewExecuter(), releaseImage, pullSecret, cacheDir)
	return r.GetReleaseArchitecture()
}
