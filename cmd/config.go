package main

import (
	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCreateConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-config",
		Short: "Generates a template of the appliance config manifest",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if err := getAssetStore().Fetch(&config.ApplianceConfig{}); err != nil {
				logrus.Fatal(err)
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			if err := deleteStateFile(rootOpts.dir); err != nil {
				logrus.Fatal(err)
			}
		},
	}

	return cmd
}
