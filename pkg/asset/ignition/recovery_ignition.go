package ignition

import (
	"fmt"
	"os"
	"path/filepath"

	configv32 "github.com/coreos/ignition/v2/config/v3_2"
	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/manifests"
	"github.com/openshift/appliance/pkg/installer"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/ignition"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
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
		interactiveUIFile := ignition.FileFromString("/etc/assisted/interactive-ui", "root", 0644, "")
		unconfiguredIgnition.Storage.Files = append(unconfiguredIgnition.Storage.Files, interactiveUIFile)

		// Explicitly disable the load-config-iso service, not required in the OVE flow
		// (even though disabled by default, the udev rule may require it).
		noConfigImageFile := ignition.FileFromString("/etc/assisted/no-config-image", "root", 0644, "")
		unconfiguredIgnition.Storage.Files = append(unconfiguredIgnition.Storage.Files, noConfigImageFile)

		_, releaseVersion, err := installerConfig.ApplianceConfig.GetRelease()
		if err != nil {
			return err
		}
  iriContent := fmt.Sprintf(`apiVersion: machineconfiguration.openshift.io/v1alpha1
kind: InternalReleaseImage
metadata:
  name: cluster
spec:
  releases:
  - name: ocp-release-bundle-%s
`, releaseVersion)

		// Keep the filepath in sync with openshift/installer#10176 until the installer min storage will be more robust.
  		iriFile := ignition.FileFromString("/etc/assisted/extra-manifests/internalreleaseimage.yaml", "root", 0644, iriContent)
        unconfiguredIgnition.Storage.Files = append(unconfiguredIgnition.Storage.Files, iriFile)
	}

	// Remove registries.conf file from unconfiguredIgnition (already added in bootstrapIgnition)
	registriesConfPath := filepath.Join(registriesConfFilePath, registriesConfFilename)
	unconfiguredIgnition.Storage.Files = funk.Filter(unconfiguredIgnition.Storage.Files, func(f igntypes.File) bool {
		return f.Path != registriesConfPath
	}).([]igntypes.File)

	i.Unconfigured = unconfiguredIgnition
	i.Bootstrap = bootstrapIgnition.Config
	i.Merged = configv32.Merge(i.Unconfigured, i.Bootstrap)

	logrus.Debug("Successfully generated recovery ignition")

	return nil
}
