package isobuilder

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/openshift/appliance/pkg/iso-builder/commands"
	"github.com/openshift/appliance/pkg/log"
)

// Run builds and executes the iso-builder CLI command tree.
func Run() error {
	var logLevel string
	var workingDir string

	rootCmd := &cobra.Command{
		Use:           filepath.Base(os.Args[0]),
		Short:         "Build an OpenShift installation ISO for disconnected environments",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetupOutputHook(logLevel)
		},
	}
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", `log level (e.g. "debug | info | warn | error")`)

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build the ISO using the embedded configuration",
		Run: func(cmd *cobra.Command, args []string) {
			builder := commands.NewBuilder(workingDir)
			if err := builder.Build(cmd.Context()); err != nil {
				logrus.Fatal(err)
			}
		},
	}
	buildCmd.Flags().StringVar(&workingDir, "working-dir", ".", "working directory for the ISO build")
	rootCmd.AddCommand(buildCmd)

	showConfigCmd := &cobra.Command{
		Use:   "show-config",
		Short: "Display the embedded configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.ShowConfig(os.Stdout)
		},
	}
	rootCmd.AddCommand(showConfigCmd)

	return rootCmd.Execute()
}
