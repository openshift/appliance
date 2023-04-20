package data

import (
	"os"
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	dataDir = "data"
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

	dataDirPath := filepath.Join(envConfig.TempDir, dataDir, templates.DataIsoFileName)
	if err := os.MkdirAll(dataDirPath, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", dataDirPath)
		return err
	}

	// 0. Check if already exists in cache

	// 1. generate mirror.sh template with bootstrap images
	// 2. use oc-mirror to download images into a temp dir
	// 3. start local registry (using registry.sh)
	// 3.1. define RegistryDataPath as TempDir/data/bootstrap_registry
	// 4. push the mirror (bootstrap images) to the registry
	// 5. kill the the registry pod

	// 6. generate mirror.sh template with all release images
	// 7. use oc-mirror to download images into a temp dir
	// 8. start local registry (using registry.sh)
	// 8.1. define RegistryDataPath as TempDir/data/release_registry
	// 9. push the mirror (release images) to the registry
	// 10. kill the the registry pod

	// genisoimage -J -r -joliet-long -o cache/dataIsoFileName dataDirPath

	dataIsoPath := filepath.Join(envConfig.CacheDir, templates.DataIsoFileName)
	a.File = &asset.File{Filename: dataIsoPath}

	f, err := os.Stat(dataIsoPath)
	if err != nil {
		return err
	}
	a.Size = f.Size()

	return nil
}

// Name returns the human-friendly name of the asset.
func (a *DataISO) Name() string {
	return "Data ISO"
}
