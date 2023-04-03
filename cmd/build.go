package main

import (
	"github.com/danielerez/openshift-appliance/pkg/asset/diskimage"
	"github.com/openshift/installer/pkg/asset"
	assetstore "github.com/openshift/installer/pkg/asset/store"
	"github.com/openshift/installer/pkg/metrics/timer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewBuildCmd() *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "build",
		Short: "build an OpenShift-based appliance disk image",
		Run: runTargetCmd([]asset.WritableAsset{
			&diskimage.ApplianceDiskImage{},
		}...),
		PostRun: func(cmd *cobra.Command, args []string) {
			deleteStateFile(rootOpts.dir)
		},
	}

	return agentCmd
}

func runTargetCmd(targets ...asset.WritableAsset) func(cmd *cobra.Command, args []string) {
	runner := func(directory string) error {
		assetStore, err := assetstore.NewStore(directory)
		if err != nil {
			return errors.Wrap(err, "failed to create asset store")
		}

		for _, a := range targets {
			err := assetStore.Fetch(a, targets...)
			if err != nil {
				err = errors.Wrapf(err, "failed to fetch %s", a.Name())
			}

			err2 := asFileWriter(a).PersistToFile(directory)
			if err2 != nil {
				err2 = errors.Wrapf(err2, "failed to write asset (%s) to disk", a.Name())
				if err != nil {
					logrus.Error(err2)
					return err
				}
				return err2
			}

			if err != nil {
				return err
			}
		}
		return nil
	}

	return func(cmd *cobra.Command, args []string) {
		timer.StartTimer(timer.TotalTimeElapsed)

		cleanup := setupFileHook(rootOpts.dir)
		defer cleanup()

		err := runner(rootOpts.dir)
		if err != nil {
			logrus.Fatal(err)
		}

		//logrus.Infof(logging.LogCreatedFiles(cmd.Name(), rootOpts.dir, targets))

		timer.StopTimer(timer.TotalTimeElapsed)
		timer.LogSummary()
	}
}

func asFileWriter(a asset.WritableAsset) asset.FileWriter {
	switch v := a.(type) {
	case asset.FileWriter:
		return v
	default:
		return asset.NewDefaultFileWriter(a)
	}
}
