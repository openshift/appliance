package appliance

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/data"
	"github.com/openshift/appliance/pkg/asset/ignition"
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
	liveIsoWorkDir = "live-iso"
	liveIsoDataDir = "registry"
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
		&recovery.BaseISO{},
		&ignition.RecoveryIgnition{},
	}
}

// Generate the appliance disk.
func (a *ApplianceLiveISO) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dataISO := &data.DataISO{}
	baseISO := &recovery.BaseISO{}
	recoveryIgnition := &ignition.RecoveryIgnition{}
	dependencies.Get(envConfig, applianceConfig, dataISO, baseISO, recoveryIgnition)

	// Build the live ISO
	if err := a.buildLiveISO(envConfig, applianceConfig, recoveryIgnition); err != nil {
		return err
	}

	applianceLiveIsoFile := filepath.Join(envConfig.AssetsDir, consts.ApplianceLiveIsoFileName)

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

func (a *ApplianceLiveISO) buildLiveISO(
	envConfig *config.EnvConfig,
	applianceConfig *config.ApplianceConfig,
	recoveryIgnition *ignition.RecoveryIgnition) error {

	// Create work dir
	workDir, err := os.MkdirTemp(envConfig.TempDir, liveIsoWorkDir)
	if err != nil {
		return err
	}

	// Create data dir
	dataDir := filepath.Join(workDir, liveIsoDataDir)
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

	// Append bootstrap ignition to initrd using isoeditor library
	sysIgnitionBytes, err := json.Marshal(recoveryIgnition.Bootstrap)
	if err != nil {
		logrus.Errorf("Failed to marshal recovery ignition to json: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}
	ignitionContent := &isoeditor.IgnitionContent{
		SystemConfigs: map[string][]byte{
			"99-bootstrap.ign": sysIgnitionBytes,
		},
	}
	initrdReader, err := isoeditor.NewInitRamFSStreamReaderFromISO(coreosIsoPath, ignitionContent)
	if err != nil {
		logrus.Errorf("Failed to create initrd with bootstrap ignition: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}
	defer func() {
		if err := initrdReader.Close(); err != nil {
			logrus.Errorf("Failed to close initrd reader: %s", err.Error())
		}
	}()

	// Write the updated initrd to the extracted ISO
	initrdPath := filepath.Join(workDir, "images/pxeboot/initrd.img")
	initrdFile, err := os.Create(initrdPath)
	if err != nil {
		logrus.Errorf("Failed to create initrd file %s: %s", initrdPath, err.Error())
		return log.StopSpinner(spinner, err)
	}
	if _, err := io.Copy(initrdFile, initrdReader); err != nil {
		if closeErr := initrdFile.Close(); closeErr != nil {
			logrus.Errorf("Failed to close initrd file: %s", closeErr.Error())
		}
		logrus.Errorf("Failed to write initrd file %s: %s", initrdPath, err.Error())
		return log.StopSpinner(spinner, err)
	}
	if err := initrdFile.Close(); err != nil {
		logrus.Errorf("Failed to close initrd file: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}

	// Embed unconfigured ignition in the extracted ISO before creating the final ISO
	ignitionBytes, err := json.Marshal(recoveryIgnition.Unconfigured)
	if err != nil {
		logrus.Errorf("Failed to marshal unconfigured ignition: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}
	if err := coreos.WriteIgnitionToExtractedISO(ignitionBytes, coreosIsoPath, workDir); err != nil {
		logrus.Errorf("Failed to write ignition to extracted ISO: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}

	// Generate live ISO
	volumeID, err := isoeditor.VolumeIdentifier(coreosIsoPath)
	if err != nil {
		return err
	}
	liveIsoFileName := filepath.Join(envConfig.AssetsDir, consts.ApplianceLiveIsoFileName)
	if err = isoeditor.Create(liveIsoFileName, workDir, volumeID); err != nil {
		logrus.Errorf("Failed to create ISO: %s", err.Error())
		return err
	}

	hybrid := syslinux.NewIsoHybrid(nil)
	if err = hybrid.Convert(liveIsoFileName); err != nil {
		logrus.Errorf("Error creating isohybrid: %s", err)
	}

	return log.StopSpinner(spinner, nil)
}
