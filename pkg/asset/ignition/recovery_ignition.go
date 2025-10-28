package ignition

import (
	"context"
	"os"

	configv32 "github.com/coreos/ignition/v2/config/v3_2"
	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/manifests"
	"github.com/openshift/appliance/pkg/installer"
	"github.com/openshift/installer/pkg/asset"
	ignasset "github.com/openshift/installer/pkg/asset/ignition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// InteractiveUIFilePath is the sentinel file path that indicates interactive UI should be enabled
	InteractiveUIFilePath = "/etc/assisted/interactive-ui"
	// InteractiveUIFileMode defines the file permissions for the interactive UI sentinel file
	InteractiveUIFileMode = 0644
	// InteractiveUIFileOwner defines the owner of the interactive UI sentinel file
	InteractiveUIFileOwner = "root"
)

// RecoveryIgnition generates the custom ignition file for the recovery ISO
type RecoveryIgnition struct {
	Unconfigured igntypes.Config
	Bootstrap    igntypes.Config
	Merged       igntypes.Config
}

var _ asset.Asset = (*RecoveryIgnition)(nil)

// Name returns the human-friendly name of the asset.
func (i *RecoveryIgnition) Name() string {
	return "Recovery Ignition"
}

// Dependencies returns dependencies used by the asset.
func (i *RecoveryIgnition) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
		&BootstrapIgnition{},
		&manifests.UnconfiguredManifests{},
	}
}

// Generate the ignition embedded in the recovery ISO.
func (i *RecoveryIgnition) Generate(_ context.Context, dependencies asset.Parents) error {
	applianceConfig := &config.ApplianceConfig{}
	envConfig := &config.EnvConfig{}
	bootstrapIgnition := &BootstrapIgnition{}
	unconfiguredManifests := &manifests.UnconfiguredManifests{}
	dependencies.Get(envConfig, applianceConfig, bootstrapIgnition, unconfiguredManifests)

	// Persists cluster-manifests required for unconfigured ignition
	if err := asset.PersistToFile(unconfiguredManifests, envConfig.TempDir); err != nil {
		return err
	}

	installerConfig := installer.InstallerConfig{
		EnvConfig:       envConfig,
		ApplianceConfig: applianceConfig,
	}
	inst := installer.NewInstaller(installerConfig)
	unconfiguredIgnitionFileName, err := inst.CreateUnconfiguredIgnition()
	if err != nil {
		return errors.Wrapf(err, "failed to create un-configured ignition")
	}

	configBytes, err := os.ReadFile(unconfiguredIgnitionFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch un-configured ignition")
	}

	unconfiguredIgnition, _, err := configv32.Parse(configBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to parse un-configured ignition")
	}

	if swag.BoolValue(installerConfig.ApplianceConfig.Config.EnableInteractiveFlow) {
		// Create an empty sentinel file to indicate that the interactive UI should be enabled.
		// The presence of this file (regardless of content) signals the system to enable interactive flow.
		interactiveUIFile := ignasset.FileFromString(InteractiveUIFilePath, InteractiveUIFileOwner, InteractiveUIFileMode, "")
		unconfiguredIgnition.Storage.Files = append(unconfiguredIgnition.Storage.Files, interactiveUIFile)
	}

	i.Unconfigured = unconfiguredIgnition
	i.Bootstrap = bootstrapIgnition.Config
	i.Merged = configv32.Merge(i.Unconfigured, i.Bootstrap)

	logrus.Debug("Successfully generated recovery ignition")

	return nil
}