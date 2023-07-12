package main

import (
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	rootOpts struct {
		dir               string
		logLevel          string
		logFile           string
		debugBootstrap    bool
		debugBaseIgnition bool
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
		NewGenerateConfigCmd(),
		NewGenerateUpgradeBundleCmd(),

		// Upgrade controller commands:
		NewStartUpgradeControllerCmd(),
		NewStartUpgradeBundleServerCmd(),
		NewStartUpgradeBundleExtractorCmd(),
		NewStartUpgradeBundleLoaderCmd(),

		// Hidden commands for debug
		NewGenerateInstallIgnitionCmd(),
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
	flags := cmd.PersistentFlags()
	flags.StringVar(
		&rootOpts.dir,
		"dir",
		".",
		"Assets directory.",
	)
	flags.StringVar(
		&rootOpts.logLevel,
		"log-level",
		"info",
		"Log level (e.g. \"debug | info | warn | error\").",
	)
	flags.StringVar(
		&rootOpts.logFile,
		"log-file",
		"",
		"Log file. The default is to create a '.openshift_appliance.log' file in the "+
			"assets directory. If the value is 'stdout' then the log will be "+
			"written to the stardard output of the process.",
	)
	return cmd
}

func runRootCmd(cmd *cobra.Command, args []string) {
	log.SetupOutputHook(rootOpts.logLevel)
}
