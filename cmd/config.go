package main

import (
	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/openshift/installer/pkg/asset"
	"github.com/spf13/cobra"
)

func NewCreateConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-config",
		Short: "Generates a template of the appliance config manifest",
		Args:  cobra.ExactArgs(0),
		Run: runTargetCmd([]asset.WritableAsset{
			&config.ApplianceConfig{},
		}...),
		PostRun: func(cmd *cobra.Command, args []string) {
			deleteStateFile(rootOpts.dir)
		},
	}

	return cmd
}
