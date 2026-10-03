package isobuilder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
	"github.com/openshift/appliance/pkg/log"
)

// RunEmbedder builds and executes the iso-config-embedder CLI command tree.
func RunEmbedder() error {
	var logLevel string
	var configPath string
	var force bool
	var outputPath string

	rootCmd := &cobra.Command{
		Use:           filepath.Base(os.Args[0]),
		Short:         "Embed and show iso-builder configuration",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetupOutputHook(logLevel)
		},
	}
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", `log level (e.g. "debug | info | warn | error")`)

	embedCmd := &cobra.Command{
		Use:   "embed [OPTIONS] <BINARY>",
		Short: "Embed a YAML config into an iso-builder binary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return EmbedConfigFile(configPath, args[0], outputPath, force)
		},
	}
	embedCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to the YAML config file [default: stdin]")
	embedCmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite an existing embedded config")
	embedCmd.Flags().StringVarP(&outputPath, "output", "o", "", "write the resulting binary to this path")
	cobra.CheckErr(embedCmd.MarkFlagRequired("output"))
	rootCmd.AddCommand(embedCmd)

	showCmd := &cobra.Command{
		Use:   "show <BINARY>",
		Short: "Show the embedded config from an iso-builder binary as YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ShowConfigFromBinary(args[0])
		},
	}
	rootCmd.AddCommand(showCmd)

	return rootCmd.Execute()
}

// EmbedConfigFile reads a YAML config and embeds it into a copy of the iso-builder binary.
func EmbedConfigFile(configPath, binaryPath, outputPath string, force bool) error {
	var data []byte
	var err error
	if configPath == "" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(configPath)
	}
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	var cfg isobuilderconfig.Config
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return fmt.Errorf("parsing YAML config: %w", err)
	}

	if cfg.PullSecret == "" {
		if authFile, ok := os.LookupEnv("PULL_SECRET_FILE"); ok && authFile != "" {
			secret, err := os.ReadFile(authFile)
			if err != nil {
				return fmt.Errorf("reading pull secret from PULL_SECRET_FILE: %w", err)
			}
			cfg.PullSecret = strings.TrimSpace(string(secret))
			logrus.Infof("Pull secret loaded from PULL_SECRET_FILE")
		}
	}

	data, err = os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("reading binary: %w", err)
	}

	if !force {
		if _, readErr := isobuilderconfig.ReadFromData(data); readErr == nil {
			return fmt.Errorf("binary already contains an embedded config; use --force to overwrite")
		}
	}

	if err := isobuilderconfig.WriteToData(data, &cfg); err != nil {
		return fmt.Errorf("embedding config: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0755); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	logrus.Infof("Configuration embedded into %s", outputPath)
	return nil
}

// ShowConfigFromBinary reads the embedded config from a binary file and prints it as YAML.
func ShowConfigFromBinary(binaryPath string) error {
	cfg, err := isobuilderconfig.ReadFromBinary(binaryPath)
	if err != nil {
		return fmt.Errorf("reading embedded config: %w", err)
	}

	cfg.PullSecret = "***REDACTED***"

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling to YAML: %w", err)
	}

	fmt.Print(string(out))
	return nil
}
