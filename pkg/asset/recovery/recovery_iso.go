package recovery

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/asset/ignition"
	"github.com/danielerez/openshift-appliance/pkg/coreos"
	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/danielerez/openshift-appliance/pkg/release"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	coreosIsoFileName   = "coreos-x86_64.iso"
	recoveryIsoFileName = "recovery.iso"
	recoveryIsoDirName  = "recovery_iso"
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
	recoveryIsoDirPath := filepath.Join(envConfig.TempDir, recoveryIsoDirName)
	c := coreos.NewCoreOS(envConfig)

	// Search for disk image in cache dir
	if fileName := c.FindInCache(recoveryIsoFileName); fileName != "" {
		logrus.Info("Reusing recovery ISO from cache...")
		a.File = &asset.File{Filename: fileName}
		generated = true
	}

	r := release.NewRelease(
		*applianceConfig.Config.OcpReleaseImage,
		applianceConfig.Config.PullSecret,
		envConfig,
	)

	if !generated {
		stop := log.Spinner("Generating recovery ISO...", "Successfully generated recovery ISO")
		defer stop()

		if err := os.MkdirAll(recoveryIsoDirPath, os.ModePerm); err != nil {
			logrus.Errorf("Failed to create dir: %s", recoveryIsoDirPath)
			return err
		}

		if err := isoeditor.Extract(coreosIsoPath, recoveryIsoDirPath); err != nil {
			logrus.Errorf("Failed to extract image: %s", err.Error())
			return err
		}

		if err := r.MirrorRelease(applianceConfig.Config.OcpReleaseImage); err != nil {
			logrus.Errorf("Failed to mirror release payload: %s", err.Error())
			return err
		}
	}

	// Embed ignition in ISO
	recoveryIsoPath := filepath.Join(envConfig.CacheDir, recoveryIsoFileName)
	ignitionBytes, err := json.Marshal(recoveryIgnition.Config)
	if err != nil {
		logrus.Errorf("Failed to marshal recovery ignition to json: %s", err.Error())
		return err
	}
	if err := c.EmbedIgnition(ignitionBytes, recoveryIsoPath); err != nil {
		logrus.Errorf("Failed to embed ignition in recovery ISO: %s", err.Error())
		return err
	}

	// Fetch ISO size
	f, err := os.Stat(recoveryIsoPath)
	if err != nil {
		logrus.Errorf("Failed to get info on recovery iso file: %s", err.Error())
		return err
	}
	a.Size = f.Size()

	a.File = &asset.File{Filename: recoveryIsoPath}

	return nil

	// TODO
	// 1. Extract base ISO - Done
	// 2. Mirror release payload - Done
	// 3. Create recovery ISO using 'isoeditor.Create' (includes extracted base ISO + release payload)
	// 4. Generate custom ignition:
	//    * On discovery - starts a local registry from tmp (with images required for bootstrap)
	//    * On installation - starts a local registry from disk (with entire payload)
	// 5. Merge custom ignition with 'un-configured ignition'
	// 6. Embed custom ignition using 'coreos-installer iso embed'
}

// Name returns the human-friendly name of the asset.
func (a *RecoveryISO) Name() string {
	return "Appliance Recovery ISO"
}
