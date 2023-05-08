package recovery

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/asset/ignition"
	"github.com/danielerez/openshift-appliance/pkg/coreos"
	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/danielerez/openshift-appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	coreosIsoFileName  = "coreos-x86_64.iso"
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
func (a *RecoveryISO) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	baseISO := &BaseISO{}
	applianceConfig := &config.ApplianceConfig{}
	recoveryIgnition := &ignition.RecoveryIgnition{}
	dependencies.Get(envConfig, baseISO, applianceConfig, recoveryIgnition)

	generated := false
	coreosIsoPath := filepath.Join(envConfig.CacheDir, coreosIsoFileName)
	recoveryIsoPath := filepath.Join(envConfig.CacheDir, templates.RecoveryIsoFileName)

	// Search for ISO in cache dir
	if fileName := envConfig.FindInCache(templates.RecoveryIsoFileName); fileName != "" {
		logrus.Info("Reusing recovery ISO from cache")
		a.File = &asset.File{Filename: fileName}
		generated = true
	}

	if !generated {
		stop := log.Spinner("Generating recovery ISO...", "Successfully generated recovery ISO")
		defer stop()

		// Copy base ISO file
		bytesRead, err := ioutil.ReadFile(coreosIsoPath)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(recoveryIsoPath, bytesRead, 0755)
		if err != nil {
			return err
		}
	}

	// Embed ignition in ISO
	c := coreos.NewCoreOS(envConfig)
	ignitionBytes, err := json.Marshal(recoveryIgnition.Config)
	if err != nil {
		logrus.Errorf("Failed to marshal recovery ignition to json: %s", err.Error())
		return err
	}
	if err := c.EmbedIgnition(ignitionBytes, recoveryIsoPath); err != nil {
		logrus.Errorf("Failed to embed ignition in recovery ISO: %s", err.Error())
		return err
	}

	return a.updateAsset(recoveryIsoPath)
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
