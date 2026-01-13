package config

import (
	"context"
	"encoding/json"
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
	CacheDir          = "cache"
	TempDir           = "temp"
	EnvConfigFileName = ".env-config.json"
)

type EnvConfig struct {
	AssetsDir string
	CacheDir  string
	TempDir   string

	IsLiveISO bool

	DebugBootstrap    bool
	DebugBaseIgnition bool
	MirrorPath        string // Path to pre-mirrored images from oc-mirror

	File *asset.File `json:"-"` // State file for persistence (excluded from JSON)
}

var _ asset.Asset = (*EnvConfig)(nil)

// Dependencies returns no dependencies.
func (e *EnvConfig) Dependencies() []asset.Asset {
	return []asset.Asset{
		&ApplianceConfig{},
	}
}

// Generate EnvConfig asset
func (e *EnvConfig) Generate(_ context.Context, dependencies asset.Parents) error {
	applianceConfig := &ApplianceConfig{}
	dependencies.Get(applianceConfig)

	if applianceConfig.File == nil {
		return errors.Errorf("Missing config file in assets directory: %s/%s", e.AssetsDir, ApplianceConfigFilename)
	}

	// Check whether the specified version is supported
	if err := e.validateOcpReleaseVersion(applianceConfig.Config.OcpRelease.Version); err != nil {
		return err
	}

	// Validate mirror-path if provided
	if e.MirrorPath != "" {
		logrus.Infof("Using pre-mirrored images from: %s", e.MirrorPath)
		if err := e.validateMirrorPath(); err != nil {
			return err
		}
	} else {
		logrus.Info("Mirror path not specified, will perform image mirroring with oc-mirror")
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

	// Serialize the EnvConfig to JSON for persistence
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal EnvConfig")
	}

	e.File = &asset.File{
		Filename: EnvConfigFileName,
		Data:     data,
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
		logrus.Warn(fmt.Sprintf("OCP release version %s is not supported. Latest supported version: %s.",
			releaseVersion, consts.MaxOcpVersion))
	}
	return nil
}

func (e *EnvConfig) validateMirrorPath() error {
	// Check if path exists
	info, err := os.Stat(e.MirrorPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Errorf("mirror-path does not exist: %s", e.MirrorPath)
		}
		return errors.Wrapf(err, "failed to access mirror-path: %s", e.MirrorPath)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return errors.Errorf("mirror-path is not a directory: %s", e.MirrorPath)
	}

	// Check if required structure exists (data subdirectory)
	dataDir := filepath.Join(e.MirrorPath, "data")
	if _, err := os.Stat(dataDir); err != nil {
		return errors.Errorf("mirror-path must contain a 'data' subdirectory (expected oc-mirror workspace structure): %s", e.MirrorPath)
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

// FindFilesInCache returns the files from cache whose name match the given regexp.
func (e *EnvConfig) FindFilesInCache(pattern string) (files []*asset.File, err error) {	
	matches, err := filepath.Glob(filepath.Join(e.CacheDir, pattern))
	if err != nil {
		return nil, err
	}

	files = make([]*asset.File, 0, len(matches))
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		filename, err := filepath.Rel(e.CacheDir, path)
		if err != nil {
			return nil, err
		}

		files = append(files, &asset.File{
			Filename: filename,
			Data:     data,
		})
	}

	return files, nil
}

// Files returns the files generated by the asset.
func (e *EnvConfig) Files() []*asset.File {
	if e.File != nil {
		return []*asset.File{e.File}
	}
	return []*asset.File{}
}

// Load returns the EnvConfig from the state file.
func (e *EnvConfig) Load(f asset.FileFetcher) (bool, error) {
	file, err := f.FetchByName(EnvConfigFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, fmt.Sprintf("failed to load %s file", EnvConfigFileName))
	}

	// Unmarshal the JSON state
	if err := json.Unmarshal(file.Data, e); err != nil {
		return false, errors.Wrapf(err, "failed to unmarshal %s", EnvConfigFileName)
	}

	e.File = file
	return true, nil
}
