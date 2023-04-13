package config

import (
	"os"
	"path/filepath"

	"github.com/openshift/installer/pkg/asset"
	"github.com/sirupsen/logrus"
)

const (
	CacheDir = "cache"
	TempDir  = "temp"
)

type EnvConfig struct {
	AssetsDir string
	CacheDir  string
	TempDir   string
}

var _ asset.Asset = (*EnvConfig)(nil)

// Dependencies returns no dependencies.
func (e *EnvConfig) Dependencies() []asset.Asset {
	return []asset.Asset{}
}

// Generate queries for the pull secret from the user.
func (e *EnvConfig) Generate(asset.Parents) error {
	e.CacheDir = filepath.Join(e.AssetsDir, CacheDir)
	e.TempDir = filepath.Join(e.AssetsDir, TempDir)

	if err := os.MkdirAll(e.CacheDir, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", e.CacheDir)
	}

	if err := os.MkdirAll(e.TempDir, os.ModePerm); err != nil {
		logrus.Errorf("Failed to create dir: %s", e.TempDir)
	}

	return nil
}

// Name returns the human-friendly name of the asset.
func (a *EnvConfig) Name() string {
	return "Env Config"
}
