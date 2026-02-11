package data

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/genisoimage"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/registry"
	"github.com/openshift/appliance/pkg/release"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	dataDir        = "data"
	dataIsoName    = "data.iso"
	dataVolumeName = "agentdata"
)

// DataISO is an asset that contains registry images
// to a recovery partition in the OpenShift-based appliance.
type DataISO struct {
	File *asset.File
	Size int64
}

var _ asset.Asset = (*DataISO)(nil)

// Dependencies returns the assets on which the Bootstrap asset depends.
func (a *DataISO) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
	}
}

// Generate the recovery ISO.
func (a *DataISO) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(envConfig, applianceConfig)

	// Search for ISO in cache dir
	if fileName := envConfig.FindInCache(consts.DataIsoFileName); fileName != "" {
		logrus.Info("Reusing data ISO from cache")
		return a.updateAsset(envConfig)
	}

	releaseConfig := release.ReleaseConfig{
		ApplianceConfig: applianceConfig,
		EnvConfig:       envConfig,
	}
	r := release.NewRelease(releaseConfig)

	dataDirPath := filepath.Join(envConfig.TempDir, dataDir)
	if err := os.MkdirAll(dataDirPath, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", dataDirPath)
		return err
	}

	spinner := log.NewSpinner(
		"Generating container registry image...",
		"Successfully generated container registry image",
		"Failed to generate container registry image",
		envConfig,
	)
	registryUri, err := registry.CopyRegistryImageIfNeeded(envConfig, applianceConfig)
	if err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	// Copying release images
	spinner = log.NewSpinner(
		fmt.Sprintf("Pulling OpenShift %s release images required for installation...",
			applianceConfig.Config.OcpRelease.Version),
		fmt.Sprintf("Successfully pulled OpenShift %s release images required for installation",
			applianceConfig.Config.OcpRelease.Version),
		fmt.Sprintf("Failed to pull OpenShift %s release images required for installation",
			applianceConfig.Config.OcpRelease.Version),
		envConfig,
	)
	registryDir, err := registry.GetRegistryDataPath(envConfig.TempDir, dataDir)
	if err != nil {
		return log.StopSpinner(spinner, err)
	}
	spinner.DirToMonitor = registryDir
	releaseImageRegistry := registry.NewRegistry(
		registry.RegistryConfig{
			DataDirPath:    registryDir,
			URI:            registryUri,
			Port:           swag.IntValue(applianceConfig.Config.ImageRegistry.Port),
			UseBinary:      swag.BoolValue(applianceConfig.Config.ImageRegistry.UseBinary),
			UseOcpRegistry: registry.ShouldUseOcpRegistry(envConfig, applianceConfig),
		})

	if err = releaseImageRegistry.StartRegistry(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = r.MirrorInstallImages(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = releaseImageRegistry.StopRegistry(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	spinner = log.NewSpinner(
		"Generating data ISO...",
		"Successfully generated data ISO",
		"Failed to generate data ISO",
		envConfig,
	)
	spinner.FileToMonitor = dataIsoName
	imageGen := genisoimage.NewGenIsoImage(nil)

	// When mirror-path is provided, copy the Docker registry data from mirror-path/data
	// to temp/data so it's in the same location as the registry container image (images/registry/registry.tar)
	registryDataSourcePath := filepath.Join(envConfig.TempDir, dataDir)
	if applianceConfig.Config.MirrorPath != nil && swag.StringValue(applianceConfig.Config.MirrorPath) != "" {
		mirrorDataPath := filepath.Join(swag.StringValue(applianceConfig.Config.MirrorPath), dataDir)
		dockerSrcPath := filepath.Join(mirrorDataPath, "docker")
		dockerDstPath := filepath.Join(registryDataSourcePath, "docker")

		logrus.Infof("Copying Docker registry data from %s to %s", dockerSrcPath, dockerDstPath)

		// Validate source directory exists
		if _, err := os.Stat(dockerSrcPath); err != nil {
			return log.StopSpinner(spinner, fmt.Errorf("docker registry data not found at %s (mirror-path may be invalid): %w", dockerSrcPath, err))
		}

		// Create destination directory
		if err := os.MkdirAll(registryDataSourcePath, os.ModePerm); err != nil {
			return log.StopSpinner(spinner, fmt.Errorf("failed to create directory for Docker registry data: %w", err))
		}

		// Copy directory recursively using cp command
		// Note: Paths are safe here as they're program-generated from validated inputs
		cpCmd := fmt.Sprintf("cp -r %s %s", dockerSrcPath, dockerDstPath)
		exec := executer.NewExecuter()
		if _, err := exec.Execute(cpCmd); err != nil {
			return log.StopSpinner(spinner, fmt.Errorf("failed to copy Docker registry data from %s to %s: %w", dockerSrcPath, dockerDstPath, err))
		}

		logrus.Infof("Successfully copied Docker registry data")
	}

	if err = imageGen.GenerateImage(envConfig.CacheDir, dataIsoName, registryDataSourcePath, dataVolumeName); err != nil {
		return log.StopSpinner(spinner, err)
	}
	return log.StopSpinner(spinner, a.updateAsset(envConfig))
}

// Name returns the human-friendly name of the asset.
func (a *DataISO) Name() string {
	return "Data ISO"
}

func (a *DataISO) updateAsset(envConfig *config.EnvConfig) error {
	dataIsoPath := filepath.Join(envConfig.CacheDir, consts.DataIsoFileName)
	a.File = &asset.File{Filename: dataIsoPath}
	f, err := os.Stat(dataIsoPath)
	if err != nil {
		return err
	}
	a.Size = f.Size()

	return nil
}
