package coreos

import (
	"encoding/json"
	"fmt"
	"github.com/cavaliergopher/grab/v3"
	"github.com/itchyny/gojq"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/release"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

const (
	templateEmbedIgnition   = "coreos-installer iso ignition embed -f --ignition-file %s %s"
	machineOsImageName      = "machine-os-images"
	coreOsFileName          = "coreos/coreos-%s.iso"
	coreOsStream            = "coreos/coreos-stream.json"
	coreOsDiskImageUrlQuery = ".architectures.x86_64.artifacts.qemu.formats[\"qcow2.gz\"].disk.location"

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

	qcowGzUrl := v.(string)
	compressed := filepath.Join(c.EnvConfig.TempDir, CoreOsDiskImageGz)
	_, err = grab.Get(compressed, qcowGzUrl)
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
		ignitionFile.Close()
		os.Remove(ignitionFile.Name())
	}()
	_, err = ignitionFile.Write(ignition)
	if err != nil {
		logrus.Errorf("Failed to write ignition data into %s: %s", ignitionFile.Name(), err.Error())
		return err
	}
	ignitionFile.Close()

	// Invoke embed ignition command
	embedCmd := fmt.Sprintf(templateEmbedIgnition, ignitionFile.Name(), isoPath)
	_, err = c.Executer.Execute(embedCmd)
	return err
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
		return nil, errors.Wrap(err, "failed to parse CoreOS stream metadata")
	}

	return m, nil
}
