package main

import (
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewGenerateConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-config",
		Short: "Generate a template of the appliance config manifest",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			configFilePath := filepath.Join(rootOpts.dir, config.ApplianceConfigFilename)
			_, err := os.Stat(configFilePath)
			if !os.IsNotExist(err) {
				logrus.Fatal("Config file already exists at assets directory")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			configAppliance := config.ApplianceConfig{}
			if err := getAssetStore().Fetch(cmd.Context(), &configAppliance); err != nil {
				logrus.Fatal(err)
			}
			if err := configAppliance.PersistToFile(rootOpts.dir); err != nil {
				logrus.Fatal(err)
			}
			logrus.Infof("Generated config file in assets directory: %s", config.ApplianceConfigFilename)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			if err := deleteStateFile(rootOpts.dir); err != nil {
				logrus.Fatal(err)
			}
		},
	}

	return cmd
}
