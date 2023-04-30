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
func (e *EnvConfig) Name() string {
	return "Env Config"
}

func (e *EnvConfig) FindInAssets(assetSubDIr, filePattern string) string {
	files, err := filepath.Glob(filepath.Join(assetSubDIr, filePattern))
	if err != nil {
		logrus.Errorf("Failed searching for file '%s' in dir '%s'", filePattern, e.CacheDir)
		return ""
	}
	if len(files) > 0 {
		return files[0]
	}
	return ""
}

func (e *EnvConfig) FindInCache(filePattern string) string {
	return e.FindInAssets(e.CacheDir, filePattern)
}

func (e *EnvConfig) FindInTemp(filePattern string) string {
	return e.FindInAssets(e.TempDir, filePattern)
}
