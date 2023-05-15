package main

import (
	"github.com/openshift/appliance/pkg/asset/appliance"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/templates"
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
	cmd.Flags().BoolVar(&rootOpts.debug, "debug", false, "")
	if err := cmd.Flags().MarkHidden("debug"); err != nil {
		logrus.Fatal(err)
	}
	return cmd
}

func runBuild(cmd *cobra.Command, args []string) {
	timer.StartTimer(timer.TotalTimeElapsed)

	cleanup := log.SetupFileHook(rootOpts.dir)
	defer cleanup()

	// Generate ApplianceDiskImage asset (including all of its dependencies)
	applianceDiskImage := appliance.ApplianceDiskImage{}
	if err := getAssetStore().Fetch(&applianceDiskImage); err != nil {
		logrus.Fatal(errors.Wrapf(err, "failed to fetch %s", applianceDiskImage.Name()))
	}

	timer.StopTimer(timer.TotalTimeElapsed)
	timer.LogSummary()

	logrus.Infof("Appliance successfully created at assets directory: %s", templates.ApplianceFileName)
}

func preRunBuild(cmd *cobra.Command, args []string) {
	// Generate EnvConfig asset
	if err := getAssetStore().Fetch(&config.EnvConfig{
		AssetsDir: rootOpts.dir,
		Debug:     rootOpts.debug,
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
