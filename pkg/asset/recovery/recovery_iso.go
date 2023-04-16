package recovery

import (
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/log"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
	"github.com/openshift/installer/pkg/asset"
)

const (
	coreosIsoFileName   = "coreos-x86_64.iso"
	recoveryIsoFileName = "recovery.iso"
	recoveryIsoDirName  = "recovery_iso"
)

// RecoveryISO is an asset that generates the bootable ISO copied
// to a recovery partition in the OpenShift-based appliance.
type RecoveryISO struct {
	File           *asset.File
	Size           int64
	LiveISOVersion string
}

var _ asset.Asset = (*RecoveryISO)(nil)

// Dependencies returns the assets on which the Bootstrap asset depends.
func (a *RecoveryISO) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&BaseISO{},
	}
}

// Generate the recovery ISO.
func (a *RecoveryISO) Generate(dependencies asset.Parents) error {
	stop := log.Spinner("Generating recovery ISO...", "Successfully generated recovery ISO")
	defer stop()

	envConfig := &config.EnvConfig{}
	baseISO := &BaseISO{}
	dependencies.Get(envConfig, baseISO)
	coreosIsoPath := filepath.Join(envConfig.CacheDir, coreosIsoFileName)
	recoveryIsoDirPath := filepath.Join(envConfig.CacheDir, recoveryIsoDirName)

	if err := os.MkdirAll(recoveryIsoDirPath, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", recoveryIsoDirPath)
		return err
	}

	if err := isoeditor.Extract(coreosIsoPath, recoveryIsoDirPath); err != nil {
		logrus.Errorf("Failed to extract image: %s", err.Error())
		return err
	}

	return nil

	//TODO: include in MGMT-14278
	//recoveryIsoPath := filepath.Join(envConfig.CacheDir, recoveryIsoFileName)
	//a.File = &asset.File{Filename: recoveryIsoPath}
	//a.LiveISOVersion = baseISO.LiveISOVersion
	//
	//f, err := os.Stat(recoveryIsoPath)
	//if err != nil {
	//	return err
	//}
	//a.Size = f.Size()



	// TODO
	// 1. Extract base ISO - Done
	// 2. Mirror release payload
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
