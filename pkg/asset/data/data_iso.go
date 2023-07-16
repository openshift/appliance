package data

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/fileutil"
	"github.com/openshift/appliance/pkg/genisoimage"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/registry"
	"github.com/openshift/appliance/pkg/release"
	"github.com/openshift/appliance/pkg/skopeo"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	dataDir            = "data"
	imagesDir          = "images"
	bootstrapMirrorDir = "data/oc-mirror/bootstrap"
	installMirrorDir   = "data/oc-mirror/install"
	dataIsoName        = "data.iso"
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

	r := release.NewRelease(*applianceConfig.Config.OcpRelease.URL, applianceConfig.Config.PullSecret, envConfig)

	dataDirPath := filepath.Join(envConfig.TempDir, dataDir)
	if err := os.MkdirAll(dataDirPath, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", dataDirPath)
		return err
	}

	spinner := log.NewSpinner(
		"Pulling container registry image...",
		"Successfully pulled container registry image",
		"Failed to pull container registry image",
		envConfig,
	)
	if err := a.copyRegistryImageIfNeeded(envConfig, applianceConfig); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err := log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	// Copying bootstrap images
	spinner = log.NewSpinner(
		fmt.Sprintf("Pulling OpenShift %s release images required for bootstrap...",
			applianceConfig.Config.OcpRelease.Version),
		fmt.Sprintf("Successfully pulled OpenShift %s release images required for bootstrap",
			applianceConfig.Config.OcpRelease.Version),
		fmt.Sprintf("Failed to pull OpenShift %s release images required for bootstrap",
			applianceConfig.Config.OcpRelease.Version),
		envConfig,
	)
	registryDir, err := registry.GetRegistryDataPath(envConfig.TempDir, bootstrapMirrorDir)
	if err != nil {
		return log.StopSpinner(spinner, err)
	}
	spinner.DirToMonitor = registryDir
	bootstrapImageRegistry := registry.NewRegistry(
		registry.RegistryConfig{
			DataDirPath: registryDir,
			URI:         swag.StringValue(applianceConfig.Config.ImageRegistry.URI),
			Port:        swag.IntValue(applianceConfig.Config.ImageRegistry.Port),
		})
	if err = bootstrapImageRegistry.StartRegistry(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = r.MirrorBootstrapImages(envConfig, applianceConfig); err != nil {
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
	registryDir, err = registry.GetRegistryDataPath(envConfig.TempDir, installMirrorDir)
	if err != nil {
		return log.StopSpinner(spinner, err)
	}
	spinner.DirToMonitor = registryDir
	releaseImageRegistry := registry.NewRegistry(
		registry.RegistryConfig{
			DataDirPath: registryDir,
			URI:         swag.StringValue(applianceConfig.Config.ImageRegistry.URI),
			Port:        swag.IntValue(applianceConfig.Config.ImageRegistry.Port),
		})

	if err = releaseImageRegistry.StartRegistry(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = r.MirrorReleaseImages(envConfig, applianceConfig); err != nil {
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
	if err = imageGen.GenerateImage(envConfig.CacheDir, dataIsoName, filepath.Join(envConfig.TempDir, dataDir)); err != nil {
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

func (a *DataISO) copyRegistryImageIfNeeded(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error {
	registryFilename := filepath.Base(consts.RegistryFilePath)
	fileInCachePath := filepath.Join(envConfig.CacheDir, registryFilename)

	// Search for registry image in cache dir
	if fileName := envConfig.FindInCache(registryFilename); fileName != "" {
		logrus.Debug("Reusing registry.tar from cache")
	} else {
		// Copying registry image to cache
		if err := skopeo.NewSkopeo(nil).CopyToFile(
			swag.StringValue(applianceConfig.Config.ImageRegistry.URI),
			consts.RegistryImageName,
			fileInCachePath); err != nil {
			return err
		}
	}

	// Copy to data dir in temp
	fileInDataDir := filepath.Join(envConfig.TempDir, dataDir, imagesDir, consts.RegistryFilePath)
	if err := fileutil.CopyFile(fileInCachePath, fileInDataDir); err != nil {
		logrus.Error(err)
		return err
	}

	return nil
}
