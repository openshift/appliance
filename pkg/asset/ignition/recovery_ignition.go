package ignition

import (
	"context"
	"os"

	configv32 "github.com/coreos/ignition/v2/config/v3_2"
	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/manifests"
	"github.com/openshift/appliance/pkg/installer"
	"github.com/openshift/installer/pkg/asset"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	i.Unconfigured = unconfiguredIgnition
	i.Bootstrap = bootstrapIgnition.Config
	i.Merged = configv32.Merge(i.Unconfigured, i.Bootstrap)

	logrus.Debug("Successfully generated recovery ignition")

	return nil
}
