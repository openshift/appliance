package data

import (
	"os"
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/fileutil"
	"github.com/danielerez/openshift-appliance/pkg/genisoimage"
	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/danielerez/openshift-appliance/pkg/registry"
	"github.com/danielerez/openshift-appliance/pkg/release"
	"github.com/danielerez/openshift-appliance/pkg/skopeo"
	"github.com/danielerez/openshift-appliance/pkg/templates"
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
	if fileName := envConfig.FindInCache(templates.DataIsoFileName); fileName != "" {
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
		"Pulling docker registry image...",
		"Successfully pulled docker registry image",
		"Failed to pull docker registry image",
		envConfig,
	)
	if err := a.copyRegistryImageIfNeeded(envConfig); err != nil {
		return log.StopSpinner(spinner, err)
	}
	log.StopSpinner(spinner, nil)

	// Copying bootstrap images
	spinner = log.NewSpinner(
		"Pulling release images required for bootstrap...",
		"Successfully pulled release images required for bootstrap",
		"Failed to pull release images required for bootstrap",
		envConfig,
	)
	registryDir := filepath.Join(envConfig.TempDir, bootstrapMirrorDir)
	spinner.DirToMonitor = registryDir
	bootstrapImageRegistry := registry.NewRegistry()
	if err := bootstrapImageRegistry.StartRegistry(registryDir); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err := r.MirrorBootstrapImages(envConfig, applianceConfig); err != nil {
		return log.StopSpinner(spinner, err)
	}
	log.StopSpinner(spinner, nil)

	// Copying release images
	spinner = log.NewSpinner(
		"Pulling release images required for installation...",
		"Successfully pulled release images required for installation",
		"Failed to pull release images required for installation",
		envConfig,
	)
	registryDir = filepath.Join(envConfig.TempDir, installMirrorDir)
	spinner.DirToMonitor = registryDir
	releaseImageRegistry := registry.NewRegistry()
	if err := releaseImageRegistry.StartRegistry(registryDir); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err := r.MirrorReleaseImages(envConfig, applianceConfig); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err := releaseImageRegistry.StopRegistry(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	log.StopSpinner(spinner, nil)

	spinner = log.NewSpinner(
		"Generating data ISO...",
		"Successfully generated data ISO",
		"Failed to generate data ISO",
		envConfig,
	)
	spinner.FileToMonitor = dataIsoName
	imageGen := genisoimage.NewGenIsoImage()
	if err := imageGen.GenerateImage(envConfig.CacheDir, dataIsoName, filepath.Join(envConfig.TempDir, dataDir)); err != nil {
		return log.StopSpinner(spinner, err)
	}
	return log.StopSpinner(spinner, a.updateAsset(envConfig))
}

// Name returns the human-friendly name of the asset.
func (a *DataISO) Name() string {
	return "Data ISO"
}

func (a *DataISO) updateAsset(envConfig *config.EnvConfig) error {
	dataIsoPath := filepath.Join(envConfig.CacheDir, templates.DataIsoFileName)
	a.File = &asset.File{Filename: dataIsoPath}
	f, err := os.Stat(dataIsoPath)
	if err != nil {
		return err
	}
	a.Size = f.Size()

	return nil
}

func (a *DataISO) copyRegistryImageIfNeeded(envConfig *config.EnvConfig) error {
	registryFilename := filepath.Base(templates.RegistryFilePath)
	fileInCachePath := filepath.Join(envConfig.CacheDir, registryFilename)

	// Search for registry image in cache dir
	if fileName := envConfig.FindInCache(registryFilename); fileName != "" {
		logrus.Debug("Reusing registry.tar from cache")
	} else {
		// Copying registry image to cache
		if err := skopeo.NewSkopeo().CopyToFile(
			templates.RegistryImage,
			templates.RegistryImageName,
			fileInCachePath); err != nil {
			return err
		}
	}

	// Copy to data dir in temp
	fileInDataDir := filepath.Join(envConfig.TempDir, dataDir, imagesDir, templates.RegistryFilePath)
	if err := fileutil.CopyFile(fileInCachePath, fileInDataDir); err != nil {
		logrus.Error(err)
		return err
	}

	return nil
}
