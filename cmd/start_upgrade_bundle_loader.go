package main

import (
	"os"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	clnt "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/appliance/pkg/upgrade"
)

// NewStartUpgradeBundleLoaderCmd creates and returns the `start-upgrade-bundle-loader` command.
func NewStartUpgradeBundleLoaderCmd() *cobra.Command {
	logger := logrus.New()
	command := &startUpgradeBundleLoaderCmd{
		logger: logrusr.New(logger),
	}
	result := &cobra.Command{
		Use:    "start-upgrade-bundle-loader",
		Short:  "Starts the program that loads the bundle into the CRI-O images directory",
		Args:   cobra.NoArgs,
		Run:    command.run,
		Hidden: true,
	}
	flags := result.Flags()
	flags.StringVar(
		&command.flags.rootDir,
		"root-dir",
		"",
		"Filesystem root. If this is specified then the rest of the paths will be "+
			"relative to it.",
	)
	flags.StringVar(
		&command.flags.nodeName,
		"node-name",
		"",
		"Name of the node where this is running.",
	)
	flags.StringVar(
		&command.flags.bundleDir,
		"bundle-dir",
		"/var/lib/upgrade",
		"Bundle directory.",
	)
	return result
}

type startUpgradeBundleLoaderCmd struct {
	logger logr.Logger
	flags  struct {
		rootDir   string
		nodeName  string
		bundleDir string
	}
}

func (c *startUpgradeBundleLoaderCmd) run(cmd *cobra.Command, argv []string) {
	// Get the context:
	ctx := cmd.Context()

	// Configure the controller runtime library to use the logger:
	ctrl.SetLogger(c.logger)

	// Check the flags:
	ok := true
	if c.flags.nodeName == "" {
		c.logger.Error(nil, "Node is mandatory")
		ok = false
	}
	if c.flags.bundleDir == "" {
		c.logger.Error(nil, "Bundle directory is mandatory")
		ok = false
	}
	if !ok {
		os.Exit(1)
	}

	// Create the API client:
	scheme := runtime.NewScheme()
	err := core.AddToScheme(scheme)
	if err != nil {
		c.logger.Error(err, "Failed to create API scheme")
		os.Exit(1)
	}
	config, err := ctrl.GetConfig()
	if err != nil {
		c.logger.Error(err, "Failed to load API configuration")
		os.Exit(1)
	}
	options := clnt.Options{
		Scheme: scheme,
	}
	client, err := clnt.New(config, options)
	if err != nil {
		c.logger.Error(err, "Failed to create API client")
		os.Exit(1)
	}

	// Start and execute the bundle loader:
	loader, err := upgrade.NewBundleExtractor().
		SetLogger(c.logger).
		SetClient(client).
		SetNode(c.flags.nodeName).
		SetBundleDir(c.flags.bundleDir).
		Build()
	if err != nil {
		c.logger.Error(err, "Failed to create loader")
		os.Exit(1)
	}
	err = loader.Run(ctx)
	if err != nil {
		c.logger.Error(err, "Failed to execute loader")
		os.Exit(1)
	}
}
