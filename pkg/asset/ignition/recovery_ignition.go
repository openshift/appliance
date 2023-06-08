package ignition

import (
	"os"

	configv32 "github.com/coreos/ignition/v2/config/v3_2"
	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/manifests"
	"github.com/openshift/appliance/pkg/installer"
	"github.com/openshift/installer/pkg/asset"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RecoveryIgnition generates the custom ignition file for the recovery ISO
type RecoveryIgnition struct {
	Config igntypes.Config
}

var _ asset.Asset = (*RecoveryIgnition)(nil)

// Name returns the human-friendly name of the asset.
func (i *RecoveryIgnition) Name() string {
	return "Recovery Ignition"
}

// Dependencies returns dependencies used by the asset.
func (i *RecoveryIgnition) Dependencies() []asset.Asset {
	return []asset.Asset{
		&BootstrapIgnition{},
		&config.EnvConfig{},
		&config.ApplianceConfig{},
		&manifests.UnconfiguredManifests{},
	}
}

// Generate the ignition embedded in the recovery ISO.
func (i *RecoveryIgnition) Generate(dependencies asset.Parents) error {
	applianceConfig := &config.ApplianceConfig{}
	envConfig := &config.EnvConfig{}
	bootstrapIgnition := &BootstrapIgnition{}
	unconfiguredManifests := &manifests.UnconfiguredManifests{}
	dependencies.Get(envConfig, applianceConfig, bootstrapIgnition, unconfiguredManifests)

	// Persists cluster-manifests required for unconfigured ignition
	if err := asset.PersistToFile(unconfiguredManifests, envConfig.TempDir); err != nil {
		return err
	}

	inst := installer.NewInstaller(envConfig)
	unconfiguredIgnitionFileName, err := inst.CreateUnconfiguredIgnition(
		swag.StringValue(applianceConfig.Config.OcpRelease.URL),
		applianceConfig.Config.PullSecret,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create un-configured ignition")
	}

	configBytes, err := os.ReadFile(unconfiguredIgnitionFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch un-configured ignition")
	}

	unconfiguredIgnitionConfig, _, err := configv32.Parse(configBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to parse un-configured ignition")
	}

	i.Config = configv32.Merge(unconfiguredIgnitionConfig, bootstrapIgnition.Config)

	logrus.Debug("Successfully generated recovery ignition")

	return nil
}
