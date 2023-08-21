package appliance

import (
	"path/filepath"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/data"
	"github.com/openshift/appliance/pkg/asset/recovery"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/openshift/assisted-service/pkg/conversions"
	"github.com/openshift/installer/pkg/asset"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ApplianceDiskImage is an asset that generates the OpenShift-based appliance.
type ApplianceDiskImage struct {
	File *asset.File
}

var _ asset.Asset = (*ApplianceDiskImage)(nil)

// Dependencies returns the assets on which the Bootstrap asset depends.
func (a *ApplianceDiskImage) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
		&BaseDiskImage{},
		&recovery.RecoveryISO{},
		&data.DataISO{},
	}
}

// Generate the appliance disk.
func (a *ApplianceDiskImage) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	recoveryISO := &recovery.RecoveryISO{}
	dataISO := &data.DataISO{}
	baseDiskImage := &BaseDiskImage{}
	dependencies.Get(envConfig, applianceConfig, recoveryISO, dataISO, baseDiskImage)

	spinner := log.NewSpinner(
		"Generating appliance disk image...",
		"Successfully generated appliance disk image",
		"Failed to generate appliance disk image",
		envConfig,
	)
	spinner.FileToMonitor = consts.ApplianceFileName

	// Render user.cfg
	if err := templates.RenderTemplateFile(
		consts.UserCfgTemplateFile,
		templates.GetUserCfgTemplateData(consts.GrubMenuEntryName, consts.GrubDefault),
		envConfig.TempDir); err != nil {
		return log.StopSpinner(spinner, err)
	}

	// Render guestfish.sh
	recoveryIsoSize := recoveryISO.Size
	dataIsoSize := dataISO.Size
	baseImageFile := baseDiskImage.File.Filename
	baseIsoSize := a.getBootPartitionsSize(baseImageFile)
	diskSize := a.getDiskSize(applianceConfig.Config.DiskSizeGB, baseIsoSize, recoveryIsoSize, dataIsoSize)

	applianceImageFile := filepath.Join(envConfig.AssetsDir, consts.ApplianceFileName)
	recoveryIsoFile := filepath.Join(envConfig.CacheDir, consts.RecoveryIsoFileName)
	dataIsoFile := filepath.Join(envConfig.CacheDir, consts.DataIsoFileName)
	cfgFile := templates.GetFilePathByTemplate(consts.UserCfgTemplateFile, envConfig.TempDir)
	gfTemplateData := templates.GetGuestfishScriptTemplateData(
		diskSize, baseIsoSize, recoveryIsoSize, dataIsoSize, baseImageFile,
		applianceImageFile, recoveryIsoFile, dataIsoFile, cfgFile)
	if err := templates.RenderTemplateFile(
		consts.GuestfishScriptTemplateFile,
		gfTemplateData,
		envConfig.TempDir); err != nil {
		return log.StopSpinner(spinner, err)
	}

	// Invoke guestfish.sh script
	logrus.Debug("Running guestfish script")
	guestfishFileName := templates.GetFilePathByTemplate(
		consts.GuestfishScriptTemplateFile, envConfig.TempDir)
	if _, err := executer.NewExecuter().Execute(guestfishFileName); err != nil {
		return log.StopSpinner(spinner, errors.Wrapf(err, "guestfish script failure"))
	}

	a.File = &asset.File{Filename: applianceImageFile}

	return log.StopSpinner(spinner, nil)
}

// Name returns the human-friendly name of the asset.
func (a *ApplianceDiskImage) Name() string {
	return "Appliance disk image"
}

func (a *ApplianceDiskImage) getDiskSize(diskSizeGB *int, baseIsoSize, recoveryIsoSize, dataIsoSize int64) int64 {
	if diskSizeGB != nil {
		return int64(*diskSizeGB)
	}

	// Calc appliance disk image size in bytes
	diskSize := baseIsoSize + recoveryIsoSize + dataIsoSize

	// Convert size to GiB (rounded up)
	return conversions.BytesToGiB(diskSize) + 1
}

func (a *ApplianceDiskImage) getBootPartitionsSize(baseImageFile string) int64 {
	partitions, err := templates.NewPartitions().GetCoreOSPartitions(baseImageFile)
	if err != nil {
		logrus.Fatal(err)
	}

	// Calc base disk image size in bytes (including an additional overhead for alignment)
	return partitions[0].Size + partitions[1].Size + partitions[2].Size + conversions.MibToBytes(1)
}
