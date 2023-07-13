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

// NewStartUpgradeBundleExtractorCmd creates and returns the `start-upgrade-bundle-extractor`
// command.
func NewStartUpgradeBundleExtractorCmd() *cobra.Command {
	logger := logrus.New()
	command := &startUpgradeBundleExtractorCmd{
		logger: logrusr.New(logger),
	}
	result := &cobra.Command{
		Use:    "start-upgrade-bundle-extractor",
		Short:  "Starts the program that downloads bundle and extracts its contents",
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
		&command.flags.bundleFile,
		"bundle-file",
		"",
		"Path of the bundle file previously copied or mounted to the node. If this "+
			"exists then it will not be necessary to download it from other nodes "+
			"of the cluster.",
	)
	flags.StringVar(
		&command.flags.bundleDir,
		"bundle-dir",
		"/var/lib/upgrade",
		"Path of the directory where the bundle will be extracted.",
	)
	flags.StringVar(
		&command.flags.bundleServer,
		"bundle-server",
		"localhost:8080",
		"Address of the server where the bundle can be downloaded from.",
	)
	return result
}

type startUpgradeBundleExtractorCmd struct {
	logger logr.Logger
	flags  struct {
		rootDir      string
		nodeName     string
		bundleFile   string
		bundleDir    string
		bundleServer string
	}
}

func (c *startUpgradeBundleExtractorCmd) run(cmd *cobra.Command, argv []string) {
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
	if c.flags.bundleFile == "" {
		c.logger.Error(nil, "Bundle file is mandatory")
		ok = false
	}
	if c.flags.bundleDir == "" {
		c.logger.Error(nil, "Bundle directory is mandatory")
		ok = false
	}
	if c.flags.bundleServer == "" {
		c.logger.Error(nil, "Bundle server is mandatory")
		ok = false
	}
	if !ok {
		os.Exit(1)
	}

	// Create the API client:
	scheme := runtime.NewScheme()
	err := core.AddToScheme(scheme)
	if err != nil {
		c.logger.Error(err, "Failed to add API scheme")
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

	// Create and run the extractor:
	extractor, err := upgrade.NewBundleExtractor().
		SetLogger(c.logger).
		SetClient(client).
		SetNode(c.flags.nodeName).
		SetRootDir(c.flags.rootDir).
		SetBundleFile(c.flags.bundleFile).
		SetBundleDir(c.flags.bundleDir).
		SetServerAddr(c.flags.bundleServer).
		Build()
	if err != nil {
		c.logger.Error(err, "Failed to create extractor")
		os.Exit(1)
	}
	err = extractor.Run(ctx)
	if err != nil {
		c.logger.Error(err, "Failed to run extractor")
		os.Exit(1)
	}
}
