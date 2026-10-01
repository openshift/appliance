package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/openshift/appliance/pkg/log"

	isobuilder "github.com/openshift/appliance/pkg/iso-builder/config"
)

var (
	rootOpts struct {
		logLevel string
	}
	embedOpts struct {
		configPath string
		force      bool
		outputPath string
	}
)

func main() {
	rootCmd := newRootCmd()
	rootCmd.AddCommand(newEmbedCmd())
	rootCmd.AddCommand(newShowCmd())

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           filepath.Base(os.Args[0]),
		Short:         "Embed and show iso-builder configuration",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetupOutputHook(rootOpts.logLevel)
		},
	}
	cmd.PersistentFlags().StringVar(&rootOpts.logLevel, "log-level", "info", "log level (e.g. \"debug | info | warn | error\")")
	return cmd
}

func newEmbedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "embed [OPTIONS] <BINARY>",
		Short: "Embed a YAML config into an iso-builder binary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEmbed(embedOpts.configPath, args[0], embedOpts.outputPath, embedOpts.force)
		},
	}
	cmd.Flags().StringVarP(&embedOpts.configPath, "config", "c", "", "path to the YAML config file [default: stdin]")
	cmd.Flags().BoolVarP(&embedOpts.force, "force", "f", false, "overwrite an existing embedded config")
	cmd.Flags().StringVarP(&embedOpts.outputPath, "output", "o", "", "write the resulting binary to this path")
	cobra.CheckErr(cmd.MarkFlagRequired("output"))
	return cmd
}

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <BINARY>",
		Short: "Show the embedded config from an iso-builder binary as YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(args[0])
		},
	}
}

// runEmbed reads a YAML config and embeds it into the iso-builder binary.
func runEmbed(configPath, binaryPath, outputPath string, force bool) error {
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

	var cfg isobuilder.Config
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return fmt.Errorf("parsing YAML config: %w", err)
	}

	if cfg.PullSecret == "" {
		if authFile, ok := os.LookupEnv("REGISTRY_AUTH_FILE"); ok && authFile != "" {
			secret, err := os.ReadFile(authFile)
			if err != nil {
				return fmt.Errorf("reading pull secret from REGISTRY_AUTH_FILE: %w", err)
			}
			cfg.PullSecret = strings.TrimSpace(string(secret))
			logrus.Infof("Pull secret loaded from REGISTRY_AUTH_FILE")
		}
	}

	if !force {
		if _, readErr := isobuilder.ReadFromBinary(binaryPath); readErr == nil {
			return fmt.Errorf("binary already contains an embedded config; use --force to overwrite")
		}
	}

	input, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("reading binary: %w", err)
	}
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("stating binary: %w", err)
	}
	if err := os.WriteFile(outputPath, input, info.Mode()); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	if err := isobuilder.WriteToBinary(outputPath, &cfg); err != nil {
		return fmt.Errorf("embedding config: %w", err)
	}

	logrus.Infof("Configuration embedded into %s", outputPath)
	return nil
}

// runShow reads the embedded config from a binary and prints it as YAML.
func runShow(binaryPath string) error {
	cfg, err := isobuilder.ReadFromBinary(binaryPath)
	if err != nil {
		return fmt.Errorf("reading embedded config: %w", err)
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling to YAML: %w", err)
	}

	fmt.Print(string(out))
	return nil
}
