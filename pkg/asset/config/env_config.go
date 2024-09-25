package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/installer/pkg/asset"
	"github.com/pkg/errors"
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

	DebugBootstrap          bool
	DebugBaseIgnition       bool
	AllowUnsupportedVersion bool
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

	if applianceConfig.File == nil {
		return errors.Errorf("Missing config file in assets directory: %s/%s", e.AssetsDir, ApplianceConfigFilename)
	}

	// Check whether the specified version is supported
	if err := e.validateOcpReleaseVersion(applianceConfig.Config.OcpRelease.Version); err != nil {
		return err
	}

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
		file := files[0]
		if !e.isValidFileSize(file) {
			return ""
		}
		return file
	}
	return ""
}

func (e *EnvConfig) isValidFileSize(file string) bool {
	f, err := os.Stat(file)
	if err != nil {
		return false
	}
	return f.Size() != 0
}

func (e *EnvConfig) validateOcpReleaseVersion(releaseVersion string) error {
	maxOcpVer, err := version.NewVersion(consts.MaxOcpVersion)
	if err != nil {
		return err
	}
	ocpVer, err := version.NewVersion(releaseVersion)
	if err != nil {
		return err
	}
	majorMinor, err := version.NewVersion(fmt.Sprintf("%d.%d", ocpVer.Segments()[0], ocpVer.Segments()[1]))
	if err != nil {
		return err
	}

	if majorMinor.GreaterThan(maxOcpVer) {
		msg := fmt.Sprintf("OCP release version %s is not supported.", releaseVersion)
		if !e.AllowUnsupportedVersion {
			logrus.Error(msg)
			logrus.Errorf("Fallback to the latest supported version: %s.", consts.MaxOcpVersion)
			logrus.Error("To override this validation use: --allow-unsupported-version")
			return errors.New(msg)
		} else {
			logrus.Warn(msg)
			logrus.Warnf("Latest supported version: %s.", consts.MaxOcpVersion)
		}
	}
	return nil
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
