package main

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	assetstore "github.com/openshift/installer/pkg/asset/store"
)

func NewCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "clean builder cache",
		Long:  "",
		Run: func(_ *cobra.Command, _ []string) {
			cleanup := setupFileHook(rootOpts.dir)
			defer cleanup()

			err := runCleanCmd(rootOpts.dir)
			if err != nil {
				logrus.Fatal(err)
			}
			logrus.Infof("Cleanup complete")
		},
	}
	return cmd
}

func runCleanCmd(directory string) error {

	deleteStateFile(directory)

	// TODO: delete cache

	return nil
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
