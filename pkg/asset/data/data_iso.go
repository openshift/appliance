package data

import (
	"os"
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/danielerez/openshift-appliance/pkg/registry"
	"github.com/danielerez/openshift-appliance/pkg/skopeo"
	"github.com/danielerez/openshift-appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	dataDir   = "data"
	imagesDir = "images"
)

var (
// TODO: use skopeo.CopyToRegistry to push AI images to local registry
//
//	aiImages = []string{
//		templates.AssistedServiceImage,
//		templates.AssistedInstallerAgentImage,
//		templates.AssistedInstallerControllerImage,
//		templates.AssistedInstallerImage,
//	}
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

	stop := log.Spinner("Generating data ISO...", "Successfully generated data ISO")
	defer stop()

	dataDirPath := filepath.Join(envConfig.TempDir, dataDir)
	if err := os.MkdirAll(dataDirPath, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", dataDirPath)
		return err
	}

	if err := a.copyRegistryImageIfNeeded(envConfig); err != nil {
		return err
	}

	// TODO: Check if already exists in cache

	// Copying registry boostrap images
	filePath := filepath.Join(envConfig.TempDir, "data/bootstrap_registry")
	bootstrapImageRegistry := registry.NewRegistry()

	if err := bootstrapImageRegistry.StartRegistry(filePath); err != nil {
		return err
	}

	// 1. generate mirror.sh template with bootstrap images
	// 2. use oc-mirror to download images into a temp dir
	// 3. start local registry (using registry.sh)
	// 3.1. define RegistryDataPath as TempDir/data/bootstrap_registry
	// 4. push the mirror (bootstrap images) to the registry
	// 5. kill the registry pod

	// 6. generate mirror.sh template with all release images
	// 7. use oc-mirror to download images into a temp dir
	// 8. start local registry (using registry.sh)
	// 8.1. define RegistryDataPath as TempDir/data/release_registry
	// 9. push the mirror (release images) to the registry
	// 10. kill the registry pod

	// genisoimage -J -joliet-long -D -V agentdata -o cache/dataIsoFileName dataDirPath

	return a.updateAsset(envConfig)
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
	imagesDirPath := filepath.Join(envConfig.TempDir, dataDir, imagesDir)

	// Search for Image in temp dir
	if fileName := envConfig.FindInTemp(filepath.Join(dataDir, imagesDir, templates.RegistryFilePath)); fileName != "" {
		logrus.Debug("Reusing registry.tar from temp")
		return nil
	}

	// Copying registry image
	err := skopeo.NewSkopeo().CopyToFile(
		templates.RegistryImage, templates.RegistryImageName, filepath.Join(imagesDirPath, templates.RegistryFilePath))
	return err
}
