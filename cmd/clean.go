package main

import (
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	assetstore "github.com/openshift/installer/pkg/asset/store"
)

func NewCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "clean assets directory (exclude builder cache)",
		Long:  "",
		Run: func(_ *cobra.Command, _ []string) {
			cleanup := log.SetupFileHook(rootOpts.dir)
			defer cleanup()

			// Remove state file
			if err := deleteStateFile(rootOpts.dir); err != nil {
				logrus.Fatal(err)
			}

			// Remove temp dir
			if err := os.RemoveAll(filepath.Join(rootOpts.dir, config.TempDir)); err != nil {
				logrus.Fatal(err)
			}

			// Remove appliance file
			if err := os.RemoveAll(filepath.Join(rootOpts.dir, templates.ApplianceFileName)); err != nil {
				logrus.Fatal(err)
			}

			logrus.Infof("Cleanup complete")
		},
	}
	return cmd
}

func deleteStateFile(directory string) error {
	store, err := assetstore.NewStore(directory)
	if err != nil {
		return errors.Wrap(err, "failed to create asset store")
	}

	err = store.DestroyState()
	if err != nil {
		return errors.Wrap(err, "failed to remove state file")
	}

	return nil
}
