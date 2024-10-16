package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/ignition"
	"github.com/openshift/appliance/pkg/asset/recovery"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/coreos"
	"github.com/openshift/appliance/pkg/fileutil"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/skopeo"
	"github.com/openshift/appliance/pkg/syslinux"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

type DeployISO struct {
	File *asset.File
}

var _ asset.Asset = (*DeployISO)(nil)

// Name returns the human-friendly name of the asset.
func (i *DeployISO) Name() string {
	return "Deployment ISO"
}

// Dependencies returns dependencies used by the asset.
func (i *DeployISO) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
		&ignition.DeployIgnition{},
		&recovery.BaseISO{},
	}
}

// Generate the base ISO.
func (i *DeployISO) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	baseISO := &recovery.BaseISO{}
	deployIgnition := &ignition.DeployIgnition{}

	dependencies.Get(envConfig, applianceConfig, baseISO, deployIgnition)

	// Search for deployment ISO in cache dir
	if fileName := envConfig.FindInAssets(consts.DeployIsoName); fileName != "" {
		logrus.Info("Configuring appliance deployment ISO")
		i.File = &asset.File{Filename: fileName}
	} else if err := i.buildDeploymentIso(envConfig, applianceConfig); err != nil {
		return err
	}

	// Embed ignition in ISO
	coreOSConfig := coreos.CoreOSConfig{
		ApplianceConfig: applianceConfig,
		EnvConfig:       envConfig,
	}
	c := coreos.NewCoreOS(coreOSConfig)
	ignitionBytes, err := json.Marshal(deployIgnition.Config)
	if err != nil {
		logrus.Errorf("Failed to marshal deploy ignition to json: %s", err.Error())
		return err
	}
	deployIsoFileName := filepath.Join(envConfig.AssetsDir, consts.DeployIsoName)
	if err = c.EmbedIgnition(ignitionBytes, deployIsoFileName); err != nil {
		logrus.Errorf("Failed to embed ignition in deploy ISO: %s", err.Error())
		return err
	}

	return nil
}

func (i *DeployISO) buildDeploymentIso(envConfig *config.EnvConfig, applianceConfig *config.ApplianceConfig) error {
	if fileName := envConfig.FindInAssets(consts.ApplianceFileName); fileName == "" {
		logrus.Infof("The appliance.raw disk image file is missing.")
		logrus.Infof("Run 'build' command for building the appliance disk image.")
		logrus.Exit(1)
		return nil
	}

	spinner := log.NewSpinner(
		"Copying appliance disk image...",
		"Successfully copied appliance disk image",
		"Failed to copy appliance disk image",
		envConfig,
	)
	deployIsoTempDir, err := os.MkdirTemp(envConfig.TempDir, consts.DeployDir)
	if err != nil {
		return err
	}
	spinner.DirToMonitor = deployIsoTempDir

	// Create deploy dir
	deployDir := filepath.Join(deployIsoTempDir, consts.DeployDir)
	if err = os.MkdirAll(deployDir, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", deployDir)
		return err
	}

	// Split appliance.raw file and output to temp dir
	// (to bypass ISO9660 limitation for large files)
	applianceImageFile := filepath.Join(envConfig.AssetsDir, consts.ApplianceFileName)
	applianceSplitFile := filepath.Join(deployDir, consts.ApplianceFileName)
	if err = fileutil.SplitFile(applianceImageFile, applianceSplitFile, "3G"); err != nil {
		logrus.Error(err)
		return err
	}

	if err = log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	// Pull appliance image into temp dir
	spinner = log.NewSpinner(
		"Pulling appliance container image...",
		"Successfully pulled appliance container image",
		"Failed to pull appliance container image",
		envConfig,
	)
	applianceTarFile := filepath.Join(deployDir, consts.ApplianceImageTar)
	if err = skopeo.NewSkopeo(nil).CopyToFile(
		consts.ApplianceImage, consts.ApplianceImageName, applianceTarFile); err != nil {
		return err
	}

	// Extract base ISO
	coreosIsoFileName := fmt.Sprintf(consts.CoreosIsoName, applianceConfig.GetCpuArchitecture())
	coreosIsoPath := filepath.Join(envConfig.CacheDir, coreosIsoFileName)
	if err = isoeditor.Extract(coreosIsoPath, deployIsoTempDir); err != nil {
		logrus.Errorf("Failed to extract ISO: %s", err.Error())
		return err
	}

	if err = log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	spinner = log.NewSpinner(
		"Generating appliance deployment ISO...",
		"Successfully generated appliance deployment ISO",
		"Failed to generate appliance deployment ISO",
		envConfig,
	)
	spinner.FileToMonitor = consts.DeployIsoName

	// Generate deployment ISO
	volumeID, err := isoeditor.VolumeIdentifier(coreosIsoPath)
	if err != nil {
		return err
	}
	deployIsoFileName := filepath.Join(envConfig.AssetsDir, consts.DeployIsoName)
	if err := isoeditor.Create(deployIsoFileName, deployIsoTempDir, volumeID); err != nil {
		logrus.Errorf("Failed to create ISO: %s", err.Error())
		return err
	}

	hybrid := syslinux.NewIsoHybrid(nil)
	if err = hybrid.Convert(deployIsoFileName); err != nil {
		logrus.Errorf("Error creating isohybrid: %s", err)
	}

	return log.StopSpinner(spinner, nil)
}
