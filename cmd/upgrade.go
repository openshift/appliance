package main

import (
	"github.com/openshift/installer/pkg/metrics/timer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/upgrade"
	"github.com/openshift/appliance/pkg/log"
)

func NewGenerateUpgradeBundleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "generate-upgrade-bundle",
		Short:  "Generates the upgrade bundle",
		PreRun: preRunGenerateUpgradeBundle,
		Run:    runGenerateUpgradeBundle,
	}
	return cmd
}

func runGenerateUpgradeBundle(cmd *cobra.Command, args []string) {
	timer.StartTimer(timer.TotalTimeElapsed)

	cleanup := log.SetupFileHook(rootOpts.dir)
	defer cleanup()

	// Fetch the asset:
	var bundle upgrade.Bundle
	err := getAssetStore().Fetch(&bundle)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to generate bundle")
	}

	timer.StopTimer(timer.TotalTimeElapsed)
	timer.LogSummary()
}

func preRunGenerateUpgradeBundle(cmd *cobra.Command, args []string) {
	err := getAssetStore().Fetch(&config.EnvConfig{
		AssetsDir: rootOpts.dir,
	})
	if err != nil {
		logrus.Fatal(err)
	}
}
