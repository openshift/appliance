package appliance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

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
	liveIsoWorkDir                = "live-iso"
	liveIsoDataDir                = "data"
	bootstrapImageName            = "/images/bootstrap-appliance.img"
	bootstrapIgnitionPath         = "/usr/lib/ignition/base.d/99-bootstrap.ign"
	defaultGrubConfigFilePath     = "EFI/redhat/grub.cfg"
	defaultIsolinuxConfigFilePath = "isolinux/isolinux.cfg"
	defaultKargsConfigFilePath    = "coreos/kargs.json"
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
func (a *ApplianceLiveISO) Generate(_ context.Context, dependencies asset.Parents) error {
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

	// Embed ignition in ISO
	coreOSConfig := coreos.CoreOSConfig{
		ApplianceConfig: applianceConfig,
		EnvConfig:       envConfig,
	}
	c := coreos.NewCoreOS(coreOSConfig)
	ignitionBytes, err := json.Marshal(recoveryIgnition.Unconfigured)
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

	// Create bootstrap.img file
	coreOSConfig := coreos.CoreOSConfig{
		ApplianceConfig: applianceConfig,
		EnvConfig:       envConfig,
	}
	c := coreos.NewCoreOS(coreOSConfig)
	ignitionBytes, err := json.Marshal(recoveryIgnition.Bootstrap)
	if err != nil {
		logrus.Errorf("Failed to marshal recovery ignition to json: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}
	bootstrapImagePath := filepath.Join(workDir, bootstrapImageName)
	if err := c.WrapIgnition(ignitionBytes, bootstrapIgnitionPath, bootstrapImagePath); err != nil {
		logrus.Errorf("Failed to create bootstrap image: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}

	// Add bootstrap.img to initrd
	replacement := fmt.Sprintf("$1 $2 %s", bootstrapImageName)
	grubCfgPath := filepath.Join(workDir, defaultGrubConfigFilePath)
	if err := editFile(grubCfgPath, `(?m)^(\s+initrd) (.+| )+$`, replacement); err != nil {
		return err
	}
	replacement = fmt.Sprintf("${1},%s ${2}", bootstrapImageName)
	isolinuxConfigFilePath := filepath.Join(workDir, defaultIsolinuxConfigFilePath)
	if err := editFile(isolinuxConfigFilePath, `(?m)^(\s+append.*initrd=\S+) (.*)$`, replacement); err != nil {
		return err
	}

	// Fix offset in kargs.json
	initrdImageOffset := int64(len(bootstrapImageName) + 1)
	if err := fixKargsOffset(workDir, defaultIsolinuxConfigFilePath, initrdImageOffset); err != nil {
		return err
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

func editFile(fileName string, reString string, replacement string) error {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(reString)
	newContent := re.ReplaceAllString(string(content), replacement)

	if err := os.WriteFile(fileName, []byte(newContent), 0600); err != nil {
		return err
	}

	return nil
}

func fixKargsOffset(workDir, configPath string, offset int64) error {
	kargsConfigFilePath := filepath.Join(workDir, defaultKargsConfigFilePath)
	kargsData, err := os.ReadFile(kargsConfigFilePath)
	if err != nil {
		return err
	}

	var kargsConfig struct {
		Default string `json:"default"`
		Files   []struct {
			End    string `json:"end"`
			Offset int64  `json:"offset"`
			Pad    string `json:"pad"`
			Path   string `json:"path"`
		} `json:"files"`
		Size int64 `json:"size"`
	}
	if err := json.Unmarshal(kargsData, &kargsConfig); err != nil {
		return err
	}
	for i, file := range kargsConfig.Files {
		if file.Path == configPath {
			kargsConfig.Files[i].Offset = file.Offset + offset
		}
	}

	workConfigFileContent, err := json.MarshalIndent(kargsConfig, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(kargsConfigFilePath, workConfigFileContent, 0600); err != nil {
		return err
	}

	return nil
}
