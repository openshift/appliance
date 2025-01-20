package main

import (
	"github.com/openshift/appliance/pkg/asset/appliance"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/deploy"
	"github.com/openshift/appliance/pkg/asset/installer"
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
	cmd.Flags().BoolVar(&buildOpts.debugBootstrap, "debug-bootstrap", false, "")
	cmd.Flags().BoolVar(&buildOpts.debugBaseIgnition, "debug-base-ignition", false, "")
	if err := cmd.Flags().MarkHidden("debug-bootstrap"); err != nil {
		logrus.Fatal(err)
	}
	if err := cmd.Flags().MarkHidden("debug-base-ignition"); err != nil {
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

	timer.StopTimer(timer.TotalTimeElapsed)
	timer.LogSummary()

	logrus.Info()
	logrus.Infof("Appliance disk image was successfully created in assets directory: %s", applianceDiskImage.File.Filename)
	logrus.Info()
	logrus.Infof("Create configuration ISO using: openshift-install agent create config-image")
	logrus.Infof("Copy openshift-install from: %s/%s", envConfig.CacheDir, "openshift-install")
	logrus.Infof("Download openshift-install from: %s", installerBinary.URL)
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
	logrus.Infof("Appliance deployment ISO is available in assets directory: %s", consts.DeployIsoName)
	logrus.Infof("Boot a machine from the ISO to initiate the deployment")
}

func preRunBuild(cmd *cobra.Command, args []string) {
	envConfig = config.EnvConfig{
		AssetsDir:         rootOpts.dir,
		DebugBootstrap:    buildOpts.debugBootstrap,
		DebugBaseIgnition: buildOpts.debugBaseIgnition,
	}

	// Generate EnvConfig asset
	if err := getAssetStore().Fetch(cmd.Context(), &envConfig); err != nil {
		logrus.Fatal(err)
	}
}

func getAssetStore() asset.Store {
	assetStore, err := assetstore.NewStore(rootOpts.dir)
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "failed to create asset store"))
	}
	return assetStore
}
