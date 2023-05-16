package main

import (
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/ignition"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewGenerateInstallIgnitionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "generate-install-ignition",
		Args:   cobra.ExactArgs(0),
		Hidden: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			if err := getAssetStore().Fetch(&config.EnvConfig{
				AssetsDir:    rootOpts.dir,
				DebugInstall: rootOpts.debugInstall,
			}); err != nil {
				logrus.Fatal(err)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			installIgnition := ignition.InstallIgnition{}
			if err := getAssetStore().Fetch(&installIgnition); err != nil {
				logrus.Fatal(err)
			}
			if err := installIgnition.PersistToFile(rootOpts.dir); err != nil {
				logrus.Fatal(err)
			}
			logrus.Infof("Generated ignition file at assets directory: %s", ignition.InstallIgnitionPath)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			if err := deleteStateFile(rootOpts.dir); err != nil {
				logrus.Fatal(err)
			}
		},
	}
	cmd.Flags().BoolVar(&rootOpts.debugInstall, "debug-install", false, "")
	if err := cmd.Flags().MarkHidden("debug-install"); err != nil {
		logrus.Fatal(err)
	}

	return cmd
}
