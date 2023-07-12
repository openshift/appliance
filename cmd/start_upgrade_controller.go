package main

import (
	"os"
	sgnl "os/signal"
	"syscall"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openshift/appliance/pkg/upgrade"
)

// NewStartUpgradeControllercmd creates and returns the `start-upgrade-controller` command.
func NewStartUpgradeControllerCmd() *cobra.Command {
	logger := logrus.New()
	command := &startUpgradeControllerCmd{
		logger: logrusr.New(logger),
	}
	result := &cobra.Command{
		Use:    "start-upgrade-controller",
		Short:  "Starts the upgrade controller",
		Args:   cobra.NoArgs,
		Run:    command.run,
		Hidden: true,
	}
	flags := result.Flags()
	flags.StringVar(
		&command.flags.namespace,
		"namespace",
		"upgrade-tool",
		"Namespace where objects will be created",
	)
	return result
}

type startUpgradeControllerCmd struct {
	logger logr.Logger
	flags  struct {
		namespace string
	}
}

func (c *startUpgradeControllerCmd) run(cmd *cobra.Command, argv []string) {
	// Get the context:
	ctx := cmd.Context()

	// Configure the controller runtime library to use the logger:
	ctrl.SetLogger(c.logger)

	// Check the flags:
	ok := true
	if c.flags.namespace == "" {
		c.logger.Error(nil, "Namespace is mandatory")
		ok = false
	}
	if !ok {
		os.Exit(1)
	}

	// Create and start the controller:
	controller, err := upgrade.NewController().
		SetLogger(c.logger).
		SetNamespace(c.flags.namespace).
		Build()
	if err != nil {
		c.logger.Error(err, "Failed to create controller")
		os.Exit(1)
	}
	err = controller.Start(ctx)
	if err != nil {
		c.logger.Error(err, "Failed to start controller")
		os.Exit(1)
	}

	// Wait for the signal to stop:
	signals := make(chan os.Signal, 1)
	sgnl.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	signal := <-signals
	c.logger.Info(
		"Received stop signal",
		"signal", signal.String(),
	)

	// Stop the controller:
	err = controller.Stop(ctx)
	if err != nil {
		c.logger.Error(err, "Failed to stop controller")
		os.Exit(1)
	}
}
