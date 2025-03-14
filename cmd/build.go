package main

import (
	"path/filepath"
	"strings"

	"github.com/openshift/appliance/pkg/asset/appliance"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/deploy"
	"github.com/openshift/appliance/pkg/asset/installer"
	"github.com/openshift/appliance/pkg/asset/upgrade"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/installer/pkg/asset"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	"github.com/openshift/installer/pkg/metrics/timer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	buildOpts struct {
		debugBootstrap    bool
		debugBaseIgnition bool
		isLiveISO         bool
	}

	envConfig    config.EnvConfig
	deployConfig *config.DeployConfig
)

func NewBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "build",
		Short:  "Build an OpenShift-based appliance disk image",
		PreRun: preRunBuild,
		Run:    runBuild,
	}
	cmd.AddCommand(getBuildISOCmd())
	cmd.AddCommand(getBuildUpgradeISOCmd())
	cmd.AddCommand(getBuildLiveISOCmd())
	cmd.PersistentFlags().BoolVar(&buildOpts.debugBootstrap, "debug-bootstrap", false, "")
	cmd.PersistentFlags().BoolVar(&buildOpts.debugBaseIgnition, "debug-base-ignition", false, "")
	if err := cmd.PersistentFlags().MarkHidden("debug-bootstrap"); err != nil {
		logrus.Fatal(err)
	}
	if err := cmd.PersistentFlags().MarkHidden("debug-base-ignition"); err != nil {
		logrus.Fatal(err)
	}
	return cmd
}

func getBuildISOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "iso",
		Short:  "Build a bootable appliance deployment ISO",
		PreRun: preRunBuild,
		Run:    runBuildISO,
	}

	deployConfig = &config.DeployConfig{}
	cmd.Flags().StringVar(&deployConfig.TargetDevice, "target-device", "/dev/sda", "Target device name to clone the appliance into")
	cmd.Flags().StringVar(&deployConfig.PostScript, "post-script", "", "Script file to invoke on completion (should be under assets directory)")
	cmd.Flags().BoolVar(&deployConfig.SparseClone, "sparse-clone", false, "Use sparse cloning - requires an empty (zeroed) device")
	cmd.Flags().BoolVar(&deployConfig.DryRun, "dry-run", false, "Skip appliance cloning (useful for getting the target device name)")

	return cmd
}

func getBuildUpgradeISOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "upgrade-iso",
		Short:  "Build an upgrade ISO",
		PreRun: preRunBuild,
		Run:    runBuildUpgradeISO,
	}
	return cmd
}

func getBuildLiveISOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "live-iso",
		Short:  "Build an appliance live ISO",
		PreRun: preRunBuildLiveISO,
		Run:    runBuildLiveISO,
	}
	return cmd
}

func runBuild(cmd *cobra.Command, args []string) {
	timer.StartTimer(timer.TotalTimeElapsed)

	cleanup := log.SetupFileHook(rootOpts.dir)
	defer cleanup()

	// Load ApplianceDiskImage asset to check whether a clean is required
	applianceDiskImage := appliance.ApplianceDiskImage{}
	if asset, err := getAssetStore().Load(&applianceDiskImage); err == nil && asset != nil {
		if asset.(*appliance.ApplianceDiskImage).File != nil {
			logrus.Infof("Appliance build flow has already been completed.")
			logrus.Infof("Run 'clean' command before re-building the appliance.")
			return
		}
	}

	// Generate ApplianceDiskImage asset (including all of its dependencies)
	if err := getAssetStore().Fetch(cmd.Context(), &applianceDiskImage); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", applianceDiskImage.Name()))
	}

	// Generate openshift-install binary download URL
	installerBinary := installer.InstallerBinary{}
	if err := getAssetStore().Fetch(cmd.Context(), &installerBinary); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", installerBinary.Name()))
	}

	// Get binary name (openshift-install or openshift-install-fips)
	installerBinaryName := applianceDiskImage.InstallerBinaryName

	timer.StopTimer(timer.TotalTimeElapsed)
	timer.LogSummary()

	logrus.Info()
	logrus.Infof("Appliance disk image was successfully created in the 'assets' directory: %s", filepath.Base(applianceDiskImage.File.Filename))
	logrus.Info()
	logrus.Infof("Create configuration ISO using: %s agent create config-image", installerBinaryName)
	logrus.Infof("Copy %s from: %s/%s", installerBinaryName, envConfig.CacheDir, installerBinaryName)
	if !strings.Contains(installerBinaryName, "fips") {
		logrus.Infof("Download %s from: %s", installerBinaryName, installerBinary.URL)
	}
}

func runBuildISO(cmd *cobra.Command, args []string) {
	cleanup := log.SetupFileHook(rootOpts.dir)
	defer cleanup()

	// Generate DeployConfig asset
	if err := getAssetStore().Fetch(cmd.Context(), deployConfig); err != nil {
		logrus.Fatal(err)
	}

	// Generate DeployISO asset
	deployISO := deploy.DeployISO{}
	if err := getAssetStore().Fetch(cmd.Context(), &deployISO); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", deployISO.Name()))
	}

	// Remove state file (cleanup)
	if err := deleteStateFile(rootOpts.dir); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info()
	logrus.Infof("Appliance deployment ISO is available in the 'assets' directory: %s", consts.DeployIsoName)
	logrus.Infof("Boot a machine from the ISO to initiate the deployment")
}

func runBuildUpgradeISO(cmd *cobra.Command, args []string) {
	cleanup := log.SetupFileHook(rootOpts.dir)
	defer cleanup()

	// Generate UpgradeConfig asset
	if err := getAssetStore().Fetch(cmd.Context(), deployConfig); err != nil {
		logrus.Fatal(err)
	}

	// Generate UpgradeISO asset
	upgradeISO := upgrade.UpgradeISO{}
	if err := getAssetStore().Fetch(cmd.Context(), &upgradeISO); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", upgradeISO.Name()))
	}

	// Remove state file (cleanup)
	if err := deleteStateFile(rootOpts.dir); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info()
	logrus.Infof("Appliance upgrade ISO is available in the 'assets' directory: %s", filepath.Base(upgradeISO.File.Filename))
	logrus.Info()
	logrus.Infof("To initiate the upgrade:")
	logrus.Infof("1. Attach the ISO to each node")
	logrus.Infof("2. oc apply -f %s", upgradeISO.UpgradeManifestFileName)
}

func runBuildLiveISO(cmd *cobra.Command, args []string) {
	timer.StartTimer(timer.TotalTimeElapsed)

	cleanup := log.SetupFileHook(rootOpts.dir)
	defer cleanup()

	// Load ApplianceLiveISO asset to check whether a clean is required
	applianceLiveISO := appliance.ApplianceLiveISO{}
	if asset, err := getAssetStore().Load(&applianceLiveISO); err == nil && asset != nil {
		if asset.(*appliance.ApplianceLiveISO).File != nil {
			logrus.Infof("Appliance build flow has already been completed.")
			logrus.Infof("Run 'clean' command before re-building the appliance.")
			return
		}
	}

	// Generate ApplianceLiveISO asset (including all of its dependencies)
	if err := getAssetStore().Fetch(cmd.Context(), &applianceLiveISO); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", applianceLiveISO.Name()))
	}

	// Generate openshift-install binary download URL
	installerBinary := installer.InstallerBinary{}
	if err := getAssetStore().Fetch(cmd.Context(), &installerBinary); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", installerBinary.Name()))
	}

	// Get binary name (openshift-install or openshift-install-fips)
	installerBinaryName := applianceLiveISO.InstallerBinaryName

	timer.StopTimer(timer.TotalTimeElapsed)
	timer.LogSummary()

	logrus.Info()
	logrus.Infof("Appliance live ISO was successfully created in the 'assets' directory: %s", filepath.Base(applianceLiveISO.File.Filename))
	logrus.Info()
	logrus.Infof("Create configuration ISO using: %s agent create config-image", installerBinaryName)
	logrus.Infof("Copy %s from: %s/%s", installerBinaryName, envConfig.CacheDir, installerBinaryName)
	if !strings.Contains(installerBinaryName, "fips") {
		logrus.Infof("Download %s from: %s", installerBinaryName, installerBinary.URL)
	}
}

func preRunBuild(cmd *cobra.Command, args []string) {
	envConfig = config.EnvConfig{
		AssetsDir:         rootOpts.dir,
		DebugBootstrap:    buildOpts.debugBootstrap,
		DebugBaseIgnition: buildOpts.debugBaseIgnition,
		IsLiveISO:         buildOpts.isLiveISO,
	}

	// Generate EnvConfig asset
	if err := getAssetStore().Fetch(cmd.Context(), &envConfig); err != nil {
		logrus.Fatal(err)
	}
}

func preRunBuildLiveISO(cmd *cobra.Command, args []string) {
	buildOpts.isLiveISO = true
	preRunBuild(cmd, args)
}

func getAssetStore() asset.Store {
	assetStore, err := assetstore.NewStore(rootOpts.dir)
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "failed to create asset store"))
	}
	return assetStore
}
