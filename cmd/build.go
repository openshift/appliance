package main

import (
	"github.com/openshift/appliance/pkg/asset/appliance"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/installer"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/installer/pkg/asset"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	"github.com/openshift/installer/pkg/metrics/timer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "build",
		Short:  "build an OpenShift-based appliance disk image",
		PreRun: preRunBuild,
		Run:    runBuild,
	}
	cmd.Flags().BoolVar(&rootOpts.debugBootstrap, "debug-bootstrap", false, "")
	cmd.Flags().BoolVar(&rootOpts.debugBaseIgnition, "debug-base-ignition", false, "")
	if err := cmd.Flags().MarkHidden("debug-bootstrap"); err != nil {
		logrus.Fatal(err)
	}
	if err := cmd.Flags().MarkHidden("debug-base-ignition"); err != nil {
		logrus.Fatal(err)
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
	if err := getAssetStore().Fetch(&applianceDiskImage); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", applianceDiskImage.Name()))
	}

	// Generate openshift-install binary download URL
	installerBinary := installer.InstallerBinary{}
	if err := getAssetStore().Fetch(&installerBinary); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", installerBinary.Name()))
	}

	timer.StopTimer(timer.TotalTimeElapsed)
	timer.LogSummary()

	logrus.Info()
	logrus.Infof("Appliance disk image was successfully created in assets directory: %s", applianceDiskImage.File.Filename)
	logrus.Info()
	logrus.Infof("Create configuration ISO using: openshift-install agent create config-image")
	logrus.Infof("Download openshift-install from: %s", installerBinary.URL)
}

func preRunBuild(cmd *cobra.Command, args []string) {
	// Generate EnvConfig asset
	if err := getAssetStore().Fetch(&config.EnvConfig{
		AssetsDir:         rootOpts.dir,
		DebugBootstrap:    rootOpts.debugBootstrap,
		DebugBaseIgnition: rootOpts.debugBaseIgnition,
	}); err != nil {
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
