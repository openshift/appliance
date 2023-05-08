package ignition

import (
	"os"

	config_32 "github.com/coreos/ignition/v2/config/v3_2"
	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/openshift/installer/pkg/asset"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// RecoveryIgnition generates the custom ignition file for the recovery ISO
type RecoveryIgnition struct {
	Config igntypes.Config
}

var _ asset.Asset = (*RecoveryIgnition)(nil)

// Name returns the human-friendly name of the asset.
func (i *RecoveryIgnition) Name() string {
	return "Recovery Ignition"
}

// Dependencies returns dependencies used by the asset.
func (i *RecoveryIgnition) Dependencies() []asset.Asset {
	return []asset.Asset{
		&BootstrapIgnition{},
	}
}

// Generate the ignition embedded in the recovery ISO.
func (i *RecoveryIgnition) Generate(dependencies asset.Parents) error {
	bootstrapIgnition := &BootstrapIgnition{}
	dependencies.Get(bootstrapIgnition)

	// Fetch un-configured ignition
	// TODO(AGENT-574): use API when ready ('openshift-install agent create unconfigured-ignition')
	//       see: https://issues.redhat.com/browse/AGENT-574
	configBytes, err := os.ReadFile("pkg/asset/ignition/unconfigured.ign")
	if err != nil {
		return errors.Wrapf(err, "failed to fetch un-configured ignition")
	}
	config, _, err := config_32.Parse(configBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to parse un-configured ignition")
	}

	i.Config = config_32.Merge(config, bootstrapIgnition.Config)

	logrus.Debug("Successfully generated recovery ignition")

	return nil
}
