package main

import (
	"os"
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewGenerateConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-config",
		Short: "generate a template of the appliance config manifest",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			configFilePath := filepath.Join(rootOpts.dir, config.ApplianceConfigFilename)
			_, error := os.Stat(configFilePath)
			if !os.IsNotExist(error) {
				logrus.Fatal("Config file already exists at assets directory")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			configAppliance := config.ApplianceConfig{}
			if err := getAssetStore().Fetch(&configAppliance); err != nil {
				logrus.Fatal(err)
			}
			if err := configAppliance.PersistToFile(rootOpts.dir); err != nil {
				logrus.Fatal(err)
			}
			logrus.Infof("Generated config file at assets directory: %s", config.ApplianceConfigFilename)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			if err := deleteStateFile(rootOpts.dir); err != nil {
				logrus.Fatal(err)
			}
		},
	}

	return cmd
}
