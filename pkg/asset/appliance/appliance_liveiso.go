package appliance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/data"
	"github.com/openshift/appliance/pkg/asset/recovery"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/coreos"
	"github.com/openshift/appliance/pkg/fileutil"
	"github.com/openshift/appliance/pkg/installer"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/syslinux"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	LiveIsoWorkDir = "live-iso"
	LiveIsoDataDir = "data"
)

// ApplianceLiveISO is an asset that generates the OpenShift-based appliance.
type ApplianceLiveISO struct {
	File                *asset.File
	InstallerBinaryName string
}

var _ asset.Asset = (*ApplianceLiveISO)(nil)

// Dependencies returns the assets on which the Bootstrap asset depends.
func (a *ApplianceLiveISO) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
		&data.DataISO{},
		&recovery.RecoveryISO{},
	}
}

// Generate the appliance disk.
func (a *ApplianceLiveISO) Generate(_ context.Context, dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dataISO := &data.DataISO{}
	recoveryISO := &recovery.RecoveryISO{}
	dependencies.Get(envConfig, applianceConfig, dataISO, recoveryISO)

	// Build the live ISO
	if err := a.buildLiveISO(envConfig, applianceConfig); err != nil {
		return err
	}

	// Embed ignition in ISO
	coreOSConfig := coreos.CoreOSConfig{
		ApplianceConfig: applianceConfig,
		EnvConfig:       envConfig,
	}
	c := coreos.NewCoreOS(coreOSConfig)
	ignitionBytes, err := json.Marshal(recoveryISO.Ignition.Config)
	if err != nil {
		logrus.Errorf("Failed to marshal recovery ignition to json: %s", err.Error())
		return err
	}
	applianceLiveIsoFile := filepath.Join(envConfig.AssetsDir, consts.ApplianceLiveIsoFileName)
	if err = c.EmbedIgnition(ignitionBytes, applianceLiveIsoFile); err != nil {
		logrus.Errorf("Failed to embed ignition in recovery ISO: %s", err.Error())
		return err
	}

	// Get installer binary
	installerConfig := installer.InstallerConfig{
		EnvConfig:       envConfig,
		ApplianceConfig: applianceConfig,
	}
	a.InstallerBinaryName = installer.NewInstaller(installerConfig).GetInstallerBinaryName()

	a.File = &asset.File{Filename: applianceLiveIsoFile}

	return nil
}

// Name returns the human-friendly name of the asset.
func (a *ApplianceLiveISO) Name() string {
	return "Appliance live ISO"
}

func (a *ApplianceLiveISO) buildLiveISO(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error {
	// Create work dir
	workDir, err := os.MkdirTemp(envConfig.TempDir, LiveIsoWorkDir)
	if err != nil {
		return err
	}

	// Create data dir
	dataDir := filepath.Join(workDir, LiveIsoDataDir)
	if err = os.MkdirAll(dataDir, os.ModePerm); err != nil {
		return err
	}

	spinner := log.NewSpinner(
		"Extracting CoreOS ISO...",
		"Successfully extracted CoreOS ISO",
		"Failed to extract CoreOS ISO",
		envConfig,
	)
	spinner.DirToMonitor = workDir

	// Extract base ISO
	coreosIsoFileName := fmt.Sprintf(consts.CoreosIsoName, applianceConfig.GetCpuArchitecture())
	coreosIsoPath := filepath.Join(envConfig.CacheDir, coreosIsoFileName)
	if err = isoeditor.Extract(coreosIsoPath, workDir); err != nil {
		logrus.Errorf("Failed to extract ISO: %s", err.Error())
		return err
	}

	if err = log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	spinner = log.NewSpinner(
		"Copying data ISO...",
		"Successfully copied data ISO",
		"Failed to copy data ISO",
		envConfig,
	)
	spinner.DirToMonitor = dataDir

	// Split data.iso file and output to work dir
	// (to bypass ISO9660 limitation for large files)
	dataIsoFile := filepath.Join(envConfig.CacheDir, consts.DataIsoFileName)
	dataIsoSplitFile := filepath.Join(dataDir, consts.DataIsoFileName)
	if err = fileutil.SplitFile(dataIsoFile, dataIsoSplitFile, "3G"); err != nil {
		logrus.Error(err)
		return err
	}

	if err = log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	spinner = log.NewSpinner(
		"Generating appliance live ISO...",
		"Successfully generated appliance live ISO",
		"Failed to generate appliance live ISO",
		envConfig,
	)
	spinner.FileToMonitor = consts.DeployIsoName

	// Generate live ISO
	volumeID, err := isoeditor.VolumeIdentifier(coreosIsoPath)
	if err != nil {
		return err
	}
	liveIsoFileName := filepath.Join(envConfig.AssetsDir, consts.ApplianceLiveIsoFileName)
	if err := isoeditor.Create(liveIsoFileName, workDir, volumeID); err != nil {
		logrus.Errorf("Failed to create ISO: %s", err.Error())
		return err
	}

	hybrid := syslinux.NewIsoHybrid(nil)
	if err = hybrid.Convert(liveIsoFileName); err != nil {
		logrus.Errorf("Error creating isohybrid: %s", err)
	}

	return log.StopSpinner(spinner, nil)
}
