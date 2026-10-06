package main

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/openshift/appliance/pkg/log"

	"go.podman.io/storage/pkg/reexec"
)

var (
	rootOpts struct {
		dir      string
		logLevel string
	}
)

func main() {
	// Required by go.podman.io/storage: the containers-storage library
	// re-executes the current process as a helper subprocess for overlay
	// operations (e.g. applying tar layers). Without this call, any code
	// path that opens a storage.Store will panic.
	if reexec.Init() {
		return
	}

	rootCmd := newRootCmd()

	for _, subCmd := range []*cobra.Command{
		NewBuildCmd(),
		NewCleanCmd(),
		NewGenerateConfigCmd(),

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
	cmd.PersistentFlags().StringVar(&rootOpts.dir, "dir", ".", "assets directory")
	cmd.PersistentFlags().StringVar(&rootOpts.logLevel, "log-level", "info", "log level (e.g. \"debug | info | warn | error\")")
	return cmd
}

func runRootCmd(cmd *cobra.Command, args []string) {
	log.SetupOutputHook(rootOpts.logLevel)
}
