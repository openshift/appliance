package main

import (
	"os"
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	rootOpts struct {
		dir      string
		logLevel string
	}
)

func main() {
	applianceMain()
}

func applianceMain() {
	rootCmd := newRootCmd()

	for _, subCmd := range []*cobra.Command{
		NewBuildCmd(),
		NewCleanCmd(),
		NewCreateConfigCmd(),
	} {
		rootCmd.AddCommand(subCmd)
	}

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error executing openshift-appliance: %v", err)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:              filepath.Base(os.Args[0]),
		Short:            "Builds an OpenShift-based appliance",
		Long:             "",
		PersistentPreRun: runRootCmd,
		SilenceErrors:    true,
		SilenceUsage:     true,
	}
	cmd.PersistentFlags().StringVar(&rootOpts.dir, "dir", ".", "assets directory")
	cmd.PersistentFlags().StringVar(&rootOpts.logLevel, "log-level", "info", "log level (e.g. \"debug | info | warn | error\")")
	return cmd
}

func runRootCmd(cmd *cobra.Command, args []string) {
	log.SetupOutputHook(rootOpts.logLevel)
}
