package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	ignasset "github.com/openshift/installer/pkg/asset/ignition"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/genisoimage"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/registry"
	"github.com/openshift/appliance/pkg/release"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	dataDir                             = "data"
	imagesDir                           = "images"
	upgradeDataDir                      = "upgrade"
	installMirrorDir                    = "upgrade/oc-mirror/install"
	upgradeVolumeNamePattern            = "upgrade_%s"
	releaseEnvFileName                  = "release.env"
	UpgradeMachineConfigFileNamePattern = "upgrade-machine-config-%s.yaml"
)

type UpgradeConfig struct {
	ReleaseImage string
}

// UpgradeISO is an asset that contains registry images for upgrade
type UpgradeISO struct {
	File                    *asset.File
	Size                    int64
	UpgradeManifestFileName string
}

var _ asset.Asset = (*UpgradeISO)(nil)

// Dependencies returns the assets on which the UpgradeISO asset depends.
func (u *UpgradeISO) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
	}
}

// Generate the upgrade ISO
func (u *UpgradeISO) Generate(_ context.Context, dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(envConfig, applianceConfig)

	releaseImage, releaseVersion, err := applianceConfig.GetRelease()
	if err != nil {
		return err
	}

	// Upgrade manifest file name
	machineConfigFileName := fmt.Sprintf(UpgradeMachineConfigFileNamePattern, releaseVersion)

	// Search for the ISO in cache dir
	upgradeISOName := fmt.Sprintf(consts.UpgradeISONamePattern, releaseVersion)
	if fileName := envConfig.FindInAssets(upgradeISOName); fileName != "" {
		logrus.Infof("Upgrade ISO already exists.")
		u.File = &asset.File{Filename: fileName}
		u.UpgradeManifestFileName = machineConfigFileName
		return nil
	}

	releaseConfig := release.ReleaseConfig{
		ApplianceConfig: applianceConfig,
		EnvConfig:       envConfig,
	}
	r := release.NewRelease(releaseConfig)

	dataDirPath := filepath.Join(envConfig.TempDir, upgradeDataDir)
	if err = os.MkdirAll(dataDirPath, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", dataDirPath)
		return err
	}

	// Create release_image file
	releaseEnv := templates.GetUpgradeISOEnv(releaseImage, releaseVersion)
	releaseImageUrlPath := filepath.Join(dataDirPath, releaseEnvFileName)
	releaseImageUrlByte := []byte(releaseEnv)
	err = os.WriteFile(releaseImageUrlPath, releaseImageUrlByte, 0600) // #nosec G306
	if err != nil {
		return err
	}

	spinner := log.NewSpinner(
		"Generating container registry image...",
		"Successfully generated container registry image",
		"Failed to generate container registry image",
		envConfig,
	)
	registryUri, err := registry.CopyRegistryImageIfNeeded(envConfig, applianceConfig)
	if err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	// Pulling release images
	spinner = log.NewSpinner(
		fmt.Sprintf("Pulling OpenShift %s release images required for upgrade...",
			applianceConfig.Config.OcpRelease.Version),
		fmt.Sprintf("Successfully pulled OpenShift %s release images required for upgrade",
			applianceConfig.Config.OcpRelease.Version),
		fmt.Sprintf("Failed to pull OpenShift %s release images required for upgrade",
			applianceConfig.Config.OcpRelease.Version),
		envConfig,
	)
	registryDir, err := registry.GetRegistryDataPath(envConfig.TempDir, installMirrorDir)
	if err != nil {
		return log.StopSpinner(spinner, err)
	}
	spinner.DirToMonitor = registryDir
	releaseImageRegistry := registry.NewRegistry(
		registry.RegistryConfig{
			DataDirPath:    registryDir,
			URI:            registryUri,
			Port:           swag.IntValue(applianceConfig.Config.ImageRegistry.Port),
			UseOcpRegistry: registry.IsUsingOcpRegistry(applianceConfig),
		})

	if err = releaseImageRegistry.StartRegistry(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = r.MirrorInstallImages(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = releaseImageRegistry.StopRegistry(); err != nil {
		return log.StopSpinner(spinner, err)
	}
	if err = log.StopSpinner(spinner, nil); err != nil {
		return err
	}

	// Generate the upgrade machine config file
	machineConfigBytes, err := u.generateUpgradeMachineConfig(releaseVersion)
	if err != nil {
		return err
	}
	machineConfigPath := filepath.Join(envConfig.AssetsDir, machineConfigFileName)
	err = os.WriteFile(machineConfigPath, machineConfigBytes, 0644) // #nosec G306
	if err != nil {
		return err
	}

	spinner = log.NewSpinner(
		"Generating upgrade ISO...",
		"Successfully generated upgrade ISO",
		"Failed to generate upgrade ISO",
		envConfig,
	)
	spinner.FileToMonitor = upgradeISOName
	imageGen := genisoimage.NewGenIsoImage(nil)
	upgradeVolumeName := fmt.Sprintf(upgradeVolumeNamePattern, releaseVersion)
	if err = imageGen.GenerateImage(envConfig.AssetsDir, upgradeISOName, filepath.Join(envConfig.TempDir, upgradeDataDir), upgradeVolumeName); err != nil {
		return log.StopSpinner(spinner, err)
	}
	upgradeIsoPath := filepath.Join(envConfig.AssetsDir, upgradeISOName)
	return log.StopSpinner(spinner, u.updateAsset(upgradeIsoPath, machineConfigFileName))
}

// Name returns the human-friendly name of the asset.
func (u *UpgradeISO) Name() string {
	return "Upgrade ISO"
}

func (u *UpgradeISO) updateAsset(upgradeIsoPath, machineConfigFileName string) error {
	u.File = &asset.File{Filename: upgradeIsoPath}
	u.UpgradeManifestFileName = machineConfigFileName
	f, err := os.Stat(upgradeIsoPath)
	if err != nil {
		return err
	}
	u.Size = f.Size()
	return nil
}

func (u *UpgradeISO) generateUpgradeMachineConfig(releaseVersion string) ([]byte, error) {
	// Generate ignition config for starting the upgrade service
	ignConfig := igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: igntypes.MaxVersion.String(),
		},
		Systemd: igntypes.Systemd{
			Units: []igntypes.Unit{{
				Name:    fmt.Sprintf("start-cluster-upgrade@%s.service", releaseVersion),
				Enabled: swag.Bool(true),
			}},
		},
	}
	ignitionRawExt, err := ignasset.ConvertToRawExtension(ignConfig)
	if err != nil {
		return nil, err
	}

	// Generate the MachineConfig with ignition config
	machineConfig := &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcfgv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "99-start-cluster-upgrade",
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": "master",
			},
		},
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignitionRawExt,
		},
	}

	return yaml.Marshal(machineConfig)
}
