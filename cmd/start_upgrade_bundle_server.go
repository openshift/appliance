package main

import (
	"os"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/upgrade"
)

// NewStartUpgradeBundleServerCmd creates and returns the `start-upgrade-bundle-server` command.
func NewStartUpgradeBundleServerCmd() *cobra.Command {
	cleanup := log.SetupFileHook(rootOpts.dir, rootOpts.logFile)
	defer cleanup()
	command := &startUpgradeBundleServerCmd{
		logger: logrusr.New(logrus.StandardLogger()),
	}
	result := &cobra.Command{
		Use:    "start-upgrade-bundle-server",
		Short:  "Starts the HTTP server that serves the upgrade bundle",
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
		&command.flags.bundleFile,
		"bundle-file",
		"",
		"Path of the bundle file previously copied or mounted to the node.",
	)
	flags.StringVar(
		&command.flags.listenAddr,
		"listen-addr",
		":8080",
		"Listen address",
	)
	return result
}

type startUpgradeBundleServerCmd struct {
	logger logr.Logger
	flags  struct {
		rootDir    string
		listenAddr string
		bundleFile string
	}
}

func (c *startUpgradeBundleServerCmd) run(cmd *cobra.Command, argv []string) {
	// Get the context:
	ctx := cmd.Context()

	// Configure the controller runtime library to use the logger:
	ctrl.SetLogger(c.logger)

	// Check the flags:
	ok := true
	if c.flags.listenAddr == "" {
		c.logger.Error(nil, "Listen address is mandatory")
		ok = false
	}
	if c.flags.bundleFile == "" {
		c.logger.Error(nil, "Bundle file is mandatory")
		ok = false
	}
	if !ok {
		os.Exit(1)
	}

	// Create and start the server:
	server, err := upgrade.NewBundleServer().
		SetLogger(c.logger).
		SetBundleFile(c.flags.bundleFile).
		SetListenAddr(c.flags.listenAddr).
		Build()
	if err != nil {
		c.logger.Error(err, "Failed to create server")
		os.Exit(1)
	}
	err = server.Run(ctx)
	if err != nil {
		c.logger.Error(err, "Failed to run server")
		os.Exit(1)
	}
}
