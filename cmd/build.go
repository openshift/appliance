package main

import (
	"github.com/danielerez/openshift-appliance/pkg/asset/appliance"
	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	"github.com/openshift/installer/pkg/metrics/timer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewBuildCmd() *cobra.Command {
	agentCmd := &cobra.Command{
		Use:    "build",
		Short:  "build an OpenShift-based appliance disk image",
		PreRun: preRun,
		Run:    run,
	}

	return agentCmd
}

func run(cmd *cobra.Command, args []string) {
	timer.StartTimer(timer.TotalTimeElapsed)

	cleanup := setupFileHook(rootOpts.dir)
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

func preRun(cmd *cobra.Command, args []string) {
	// Generate EnvConfig asset
	if err := getAssetStore().Fetch(&config.EnvConfig{AssetsDir: rootOpts.dir}); err != nil {
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
