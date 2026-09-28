package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	appliancedata "github.com/openshift/appliance/data"
	"github.com/openshift/appliance/pkg/log"

	isobuilder "github.com/openshift/appliance/pkg/iso-builder"
	installerdata "github.com/openshift/installer/data"
)

var (
	rootOpts struct {
		logLevel string
	}
	buildOpts struct {
		workingDir string
	}
)

func main() {
	// Embed the data/ directory (systemd units, scripts, udev rules) into
	// the binary so iso-builder runs standalone without needing the repo
	// layout on disk.
	installerdata.Assets = http.FS(appliancedata.EmbeddedAssets)

	rootCmd := newRootCmd()
	rootCmd.AddCommand(newBuildCmd())

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           filepath.Base(os.Args[0]),
		Short:         "Build an OpenShift installation ISO for disconnected environments",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetupOutputHook(rootOpts.logLevel)
		},
	}
	cmd.PersistentFlags().StringVar(&rootOpts.logLevel, "log-level", "info", "log level (e.g. \"debug | info | warn | error\")")
	return cmd
}

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the ISO using the embedded configuration",
		Run: func(cmd *cobra.Command, args []string) {
			builder := isobuilder.NewBuilder(buildOpts.workingDir)
			if err := builder.Build(cmd.Context()); err != nil {
				logrus.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVar(&buildOpts.workingDir, "working-dir", ".", "working directory for the ISO build")
	return cmd
}
