package config

import (
	"fmt"
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

	DebugBootstrap bool
	DebugInstall   bool
}

var _ asset.Asset = (*EnvConfig)(nil)

// Dependencies returns no dependencies.
func (e *EnvConfig) Dependencies() []asset.Asset {
	return []asset.Asset{
		&ApplianceConfig{},
	}
}

// Generate EnvConfig asset
func (e *EnvConfig) Generate(dependencies asset.Parents) error {
	applianceConfig := &ApplianceConfig{}
	dependencies.Get(applianceConfig)

	// Cache dir in 'version-arch' format
	cacheDirPattern := fmt.Sprintf("%s-%s",
		applianceConfig.Config.OcpRelease.Version, applianceConfig.GetCpuArchitecture())

	e.CacheDir = filepath.Join(e.AssetsDir, CacheDir, cacheDirPattern)
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

func (e *EnvConfig) findInDir(dir, filePattern string) string {
	files, err := filepath.Glob(filepath.Join(dir, filePattern))
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
	return e.findInDir(e.CacheDir, filePattern)
}

func (e *EnvConfig) FindInTemp(filePattern string) string {
	return e.findInDir(e.TempDir, filePattern)
}

func (e *EnvConfig) FindInAssets(filePattern string) string {
	if file := e.FindInCache(filePattern); file != "" {
		return file
	}
	if file := e.FindInTemp(filePattern); file != "" {
		return file
	}
	return e.findInDir(e.AssetsDir, filePattern)
}
