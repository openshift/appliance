package ignition

import (
	"context"
	"fmt"
	"os"
	"strings"

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
		interactiveUIFile := ignition.FileFromString("/etc/assisted/interactive-ui", "root", 0644, "")
		unconfiguredIgnition.Storage.Files = append(unconfiguredIgnition.Storage.Files, interactiveUIFile)

		// Explicitly disable the load-config-iso service, not required in the OVE flow
		// (even though disabled by default, the udev rule may require it).
		noConfigImageFile := ignition.FileFromString("/etc/assisted/no-config-image", "root", 0644, "")
		unconfiguredIgnition.Storage.Files = append(unconfiguredIgnition.Storage.Files, noConfigImageFile)

		version := getOCPVersion(installerConfig.ApplianceConfig)
  irrContent := fmt.Sprintf(`apiVersion: machineconfiguration.openshift.io/v1alpha1
  kind: InternalReleaseImage
  metadata:
    name: cluster
  spec:
    releases:
    - name: ocp-release-bundle-%s
  `, version)

        irrFile := ignition.FileFromString("/etc/assisted/manifests/internalreleaseimage.yaml", "root", 0644, irrContent)
        unconfiguredIgnition.Storage.Files = append(unconfiguredIgnition.Storage.Files, irrFile)
	}

	i.Unconfigured = unconfiguredIgnition
	i.Bootstrap = bootstrapIgnition.Config
	i.Merged = configv32.Merge(i.Unconfigured, i.Bootstrap)

	logrus.Debug("Successfully generated recovery ignition")

	return nil
}

// getOCPVersion returns the OpenShift version string for the InternalReleaseImage.
// If a release URL is provided, it extracts the version tag from the URL.
// Otherwise, it constructs the version from the configured version and architecture.
  func getOCPVersion(applianceConfig *config.ApplianceConfig) string {
        if applianceConfig.Config.OcpRelease.URL != nil && *applianceConfig.Config.OcpRelease.URL != "" {
                return extractVersionFromURL(*applianceConfig.Config.OcpRelease.URL)
        }
        ocpVersion := applianceConfig.Config.OcpRelease.Version
        arch := *applianceConfig.Config.OcpRelease.CpuArchitecture
        return fmt.Sprintf("%s-%s", ocpVersion, arch)
  }

  // extractVersionFromURL extracts the version tag from a container image URL.
  func extractVersionFromURL(url string) string {
        parts := strings.Split(url, ":")
        if len(parts) >= 2 {
                return parts[len(parts)-1]
        }
        return ""
  }