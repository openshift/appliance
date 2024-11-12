package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/ignition"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/coreos"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	recoveryIsoDirName = "recovery_iso"
)

// RecoveryISO is an asset that generates the bootable ISO copied
// to a recovery partition in the OpenShift-based appliance.
type RecoveryISO struct {
	File *asset.File
	Size int64
}

var _ asset.Asset = (*RecoveryISO)(nil)

// Dependencies returns the assets on which the Bootstrap asset depends.
func (a *RecoveryISO) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
		&ignition.RecoveryIgnition{},
		&BaseISO{},
	}
}

// Generate the recovery ISO.
func (a *RecoveryISO) Generate(_ context.Context, dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	baseISO := &BaseISO{}
	applianceConfig := &config.ApplianceConfig{}
	recoveryIgnition := &ignition.RecoveryIgnition{}
	dependencies.Get(envConfig, baseISO, applianceConfig, recoveryIgnition)

	coreosIsoFileName := fmt.Sprintf(coreosIsoName, applianceConfig.GetCpuArchitecture())
	coreosIsoPath := filepath.Join(envConfig.CacheDir, coreosIsoFileName)
	recoveryIsoPath := filepath.Join(envConfig.CacheDir, consts.RecoveryIsoFileName)
	recoveryIsoDirPath := filepath.Join(envConfig.TempDir, recoveryIsoDirName)

	var spinner *log.Spinner

	// Search for ISO in cache dir
	if fileName := envConfig.FindInCache(consts.RecoveryIsoFileName); fileName != "" {
		logrus.Info("Reusing recovery CoreOS ISO from cache")
		a.File = &asset.File{Filename: fileName}
	} else {
		spinner = log.NewSpinner(
			"Generating recovery CoreOS ISO...",
			"Successfully generated recovery CoreOS ISO",
			"Failed to generate recovery CoreOS ISO",
			envConfig,
		)
		spinner.FileToMonitor = consts.RecoveryIsoFileName

		// Extracting the base ISO and generating the recovery ISO with a different volume label ('agentboot').
		if err := os.MkdirAll(recoveryIsoDirPath, os.ModePerm); err != nil {
			logrus.Errorf("Failed to create dir: %s", recoveryIsoDirPath)
			return err
		}
		if err := isoeditor.Extract(coreosIsoPath, recoveryIsoDirPath); err != nil {
			logrus.Errorf("Failed to extract ISO: %s", err.Error())
			return err
		}
		if err := isoeditor.Create(recoveryIsoPath, recoveryIsoDirPath, consts.RecoveryPartitionName); err != nil {
			logrus.Errorf("Failed to create ISO: %s", err.Error())
			return err
		}
	}

	// Embed ignition in ISO
	coreOSConfig := coreos.CoreOSConfig{
		ApplianceConfig: applianceConfig,
		EnvConfig:       envConfig,
	}
	c := coreos.NewCoreOS(coreOSConfig)
	ignitionBytes, err := json.Marshal(recoveryIgnition.Config)
	if err != nil {
		logrus.Errorf("Failed to marshal recovery ignition to json: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}
	if err = c.EmbedIgnition(ignitionBytes, recoveryIsoPath); err != nil {
		logrus.Errorf("Failed to embed ignition in recovery ISO: %s", err.Error())
		return log.StopSpinner(spinner, err)
	}

	return log.StopSpinner(spinner, a.updateAsset(recoveryIsoPath))
}

// Name returns the human-friendly name of the asset.
func (a *RecoveryISO) Name() string {
	return "Appliance Recovery ISO"
}

func (a *RecoveryISO) updateAsset(recoveryIsoPath string) error {
	a.File = &asset.File{Filename: recoveryIsoPath}
	f, err := os.Stat(recoveryIsoPath)
	if err != nil {
		return err
	}
	a.Size = f.Size()

	return nil
}
