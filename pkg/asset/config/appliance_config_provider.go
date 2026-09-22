package config

import (
	"github.com/openshift/appliance/pkg/types"
	"github.com/openshift/installer/pkg/asset"
)

// ApplianceConfigProvider allows callers to inject a pre-built
// configuration into the asset dependency graph. When Config is
// populated, ApplianceConfig.Generate() uses it instead of
// producing a sample template. When Config is nil (default),
// the normal disk-based flow is unaffected.
type ApplianceConfigProvider struct {
	Config *types.ApplianceConfig
}

var _ asset.WritableAsset = (*ApplianceConfigProvider)(nil)

func (*ApplianceConfigProvider) Name() string {
	return "Appliance Config Provider"
}

func (*ApplianceConfigProvider) Dependencies() []asset.Asset {
	return nil
}

func (a *ApplianceConfigProvider) Generate(_ asset.Parents) error {
	return nil
}

func (a *ApplianceConfigProvider) Load(_ asset.FileFetcher) (bool, error) {
	return false, nil
}

func (a *ApplianceConfigProvider) Files() []*asset.File {
	return []*asset.File{}
}
