package coreos

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cavaliergopher/grab/v3"
	"github.com/itchyny/gojq"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/release"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
	"github.com/sirupsen/logrus"
)

const (
	templateEmbedIgnition   = "coreos-installer iso ignition embed -f --ignition-file %s %s"
	machineOsImageName      = "machine-os-images"
	coreOsFileName          = "coreos/coreos-%s.iso"
	coreOsStream            = "coreos/coreos-stream.json"
	coreOsDiskImageUrlQuery = ".architectures.x86_64.artifacts.metal.formats[\"raw.gz\"].disk.location"

	CoreOsDiskImageGz = "coreos.tar.gz"
)

type CoreOS interface {
	DownloadDiskImage() (string, error)
	DownloadISO() (string, error)
	EmbedIgnition(ignition []byte, isoPath string) error
	FetchCoreOSStream() (map[string]any, error)
}

type CoreOSConfig struct {
	Executer        executer.Executer
	EnvConfig       *config.EnvConfig
	Release         release.Release
	ApplianceConfig *config.ApplianceConfig
}

type coreos struct {
	CoreOSConfig
}

func NewCoreOS(config CoreOSConfig) CoreOS {
	if config.Executer == nil {
		config.Executer = executer.NewExecuter()
	}
	if config.Release == nil {
		releaseConfig := release.ReleaseConfig{
			ApplianceConfig: config.ApplianceConfig,
			EnvConfig:       config.EnvConfig,
		}
		config.Release = release.NewRelease(releaseConfig)
	}

	return &coreos{
		CoreOSConfig: config,
	}
}

func (c *coreos) DownloadDiskImage() (string, error) {
	coreosStream, err := c.FetchCoreOSStream()
	if err != nil {
		return "", err
	}
	query, err := gojq.Parse(coreOsDiskImageUrlQuery)
	if err != nil {
		return "", err
	}
	iter := query.Run(coreosStream)
	v, ok := iter.Next()
	if !ok {
		return "", err
	}

	rawGzUrl := v.(string)
	compressed := filepath.Join(c.EnvConfig.TempDir, CoreOsDiskImageGz)
	_, err = grab.Get(compressed, rawGzUrl)
	if err != nil {
		return "", err
	}

	return compressed, nil
}

func (c *coreos) DownloadISO() (string, error) {
	fileName := fmt.Sprintf(coreOsFileName, c.ApplianceConfig.GetCpuArchitecture())
	return c.Release.ExtractFile(machineOsImageName, fileName)
}

func (c *coreos) EmbedIgnition(ignition []byte, isoPath string) error {
	// Write ignition to a temporary file
	ignitionFile, err := os.CreateTemp(c.EnvConfig.TempDir, "config.ign")
	if err != nil {
		return err
	}
	defer func() {
		if err := ignitionFile.Close(); err != nil {
			logrus.Errorf("Failed to close ignition file: %s", err.Error())
			return
		}
		if err := os.Remove(ignitionFile.Name()); err != nil {
			logrus.Errorf("Failed to remove ignition file: %s", err.Error())
		}
	}()
	_, err = ignitionFile.Write(ignition)
	if err != nil {
		logrus.Errorf("Failed to write ignition data into %s: %s", ignitionFile.Name(), err.Error())
		return err
	}

	// Invoke embed ignition command
	embedCmd := fmt.Sprintf(templateEmbedIgnition, ignitionFile.Name(), isoPath)
	_, err = c.Executer.Execute(embedCmd)
	return err
}

// WriteIgnitionToExtractedISO writes ignition content to an already-extracted ISO directory.
// This should be called before isoeditor.Create() to avoid a redundant Extract/Create cycle.
func WriteIgnitionToExtractedISO(ignition []byte, isoPath string, extractedDir string) error {
	// Get the ignition image with embedded ignition content
	ignitionContent := &isoeditor.IgnitionContent{
		Config: ignition,
	}
	fileData, err := isoeditor.NewIgnitionImageReader(isoPath, ignitionContent)
	if err != nil {
		return fmt.Errorf("failed to create ignition image: %w", err)
	}

	// Write the returned files to the extracted directory
	var errs []error
	for _, fd := range fileData {
		defer func(data io.ReadCloser, filename string) {
			if err := data.Close(); err != nil {
				logrus.Errorf("Failed to close data for %s: %s", filename, err.Error())
			}
		}(fd.Data, fd.Filename)

		filePath := filepath.Join(extractedDir, fd.Filename)
		file, err := os.Create(filePath)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		defer func(f *os.File) {
			if err := f.Close(); err != nil {
				logrus.Errorf("Failed to close file %s: %s", f.Name(), err.Error())
			}
		}(file)

		_, err = io.Copy(file, fd.Data)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (c *coreos) FetchCoreOSStream() (map[string]any, error) {
	path, err := c.Release.ExtractFile(machineOsImageName, coreOsStream)
	if err != nil {
		return nil, err
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err = json.Unmarshal(file, &m); err != nil {
		return nil, fmt.Errorf("failed to parse CoreOS stream metadata: %w", err)
	}

	return m, nil
}
