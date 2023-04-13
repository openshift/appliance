package appliance

import (
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/asset/recovery"
	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/danielerez/openshift-appliance/pkg/templates"
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
		&recovery.RecoveryISO{},
		&BaseDiskImage{},
	}
}

// Generate the appliance disk.
func (a *ApplianceDiskImage) Generate(dependencies asset.Parents) error {
	logrus.Info("Generating appliance disk image...")

	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	recoveryISO := &recovery.RecoveryISO{}
	baseDiskImage := &BaseDiskImage{}
	dependencies.Get(envConfig, applianceConfig, recoveryISO, baseDiskImage)

	// Render user.cfg
	if err := templates.RenderTemplateFile(
		templates.UserCfgTemplateFile,
		templates.GetUserCfgTemplateData(recoveryISO.LiveISOVersion),
		envConfig.TempDir); err != nil {
		return err
	}

	// Render guestfish.sh
	diskSize := int64(applianceConfig.Config.DiskSizeGB)
	recoveryPartitionSize := recoveryISO.Size
	baseImageFile := baseDiskImage.File.Filename
	applianceImageFile := filepath.Join(envConfig.AssetsDir, templates.ApplianceFileName)
	recoveryIsoFile := filepath.Join(envConfig.CacheDir, templates.RecoveryIsoFileName)
	cfgFile := templates.GetFilePathByTemplate(templates.UserCfgTemplateFile, envConfig.TempDir)
	gfTemplateData := templates.GetGuestfishScriptTemplateData(
		diskSize, recoveryPartitionSize, baseImageFile,
		applianceImageFile, recoveryIsoFile, cfgFile)
	if err := templates.RenderTemplateFile(
		templates.GuestfishScriptTemplateFile,
		gfTemplateData,
		envConfig.TempDir); err != nil {
		return err
	}

	// Invoke guestfish.sh script
	logrus.Debug("Running guestfish script")
	guestfishFileName := templates.GetFilePathByTemplate(
		templates.GuestfishScriptTemplateFile, envConfig.TempDir)
	if _, err := executer.NewExecuter().Execute(guestfishFileName); err != nil {
		return errors.Wrapf(err, "guestfish script failure")
	}

	a.File = &asset.File{Filename: applianceImageFile}

	return nil
}

// Name returns the human-friendly name of the asset.
func (a *ApplianceDiskImage) Name() string {
	return "Appliance disk image"
}
