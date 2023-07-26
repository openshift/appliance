package ignitionutil

import (
	"encoding/json"

	ignitionConfig "github.com/coreos/ignition/v2/config/v3_2"
	"github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/coreos/ignition/v2/config/validate"
	"github.com/openshift/appliance/pkg/fileutil"
	"github.com/pkg/errors"
)

//go:generate mockgen -source=ignition.go -package=ignitionutil -destination=mock_ignition.go
type Ignition interface {
	ParseIgnitionFile(path string) (*types.Config, error)
	WriteIgnitionFile(path string, config *types.Config) error
	MergeIgnitionConfig(base *types.Config, overrides *types.Config) (*types.Config, error)
}

type IgnitionConfig struct {
	OSInterface fileutil.OSInterface
}

type ignition struct {
	IgnitionConfig
}

func NewIgnition(config IgnitionConfig) Ignition {
	if config.OSInterface == nil {
		config.OSInterface = &fileutil.OSFS{}
	}
	return &ignition{
		IgnitionConfig: config,
	}
}

// ParseIgnitionFile reads an ignition config from a given path on disk
func (i *ignition) ParseIgnitionFile(path string) (*types.Config, error) {
	configBytes, err := i.OSInterface.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading file %s", path)
	}
	configLatest, _, err := ignitionConfig.Parse(configBytes)
	return &configLatest, err
}

// WriteIgnitionFile writes an ignition config to a given path on disk
func (i *ignition) WriteIgnitionFile(path string, config *types.Config) error {
	updatedBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = i.OSInterface.WriteFile(path, updatedBytes, 0600)
	if err != nil {
		return errors.Wrapf(err, "error writing file %s", path)
	}
	return nil
}

// MergeIgnitionConfig merges the specified configs and check the result is a valid ignition config
func (i *ignition) MergeIgnitionConfig(base *types.Config, overrides *types.Config) (*types.Config, error) {
	config := ignitionConfig.Merge(*base, *overrides)
	report := validate.ValidateWithContext(config, nil)
	if report.IsFatal() {
		return &config, errors.Errorf("merged ignition config is invalid: %s", report.String())
	}
	return &config, nil
}
