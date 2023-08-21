package config

import (
	"github.com/openshift/installer/pkg/asset"
)

type DeployConfig struct {
	TargetDevice string
	PostScript   string
	SparseClone  bool
	DryRun       bool
}

var _ asset.Asset = (*DeployConfig)(nil)

// Dependencies returns no dependencies.
func (e *DeployConfig) Dependencies() []asset.Asset {
	return []asset.Asset{
	}
}

// Generate EnvConfig asset
func (e *DeployConfig) Generate(dependencies asset.Parents) error {
	return nil
}

// Name returns the human-friendly name of the asset.
func (e *DeployConfig) Name() string {
	return "Deploy Config"
}
