package coreos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cavaliergopher/grab/v3"
	"github.com/itchyny/gojq"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/release"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	DownloadDiskImage(releaseImage, pullSecret string) (string, error)
	DownloadISO(releaseImage, cpuArch, pullSecret string) (string, error)
	EmbedIgnition(ignition []byte, isoPath string) error
	FetchCoreOSStream(releaseImage, pullSecret string) (map[string]any, error)
}

type coreos struct {
	EnvConfig *config.EnvConfig
}

func NewCoreOS(envConfig *config.EnvConfig) CoreOS {
	return &coreos{
		EnvConfig: envConfig,
	}
}

func (c *coreos) DownloadDiskImage(releaseImage, pullSecret string) (string, error) {
	coreosStream, err := c.FetchCoreOSStream(releaseImage, pullSecret)
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

func (c *coreos) DownloadISO(releaseImage, cpuArch, pullSecret string) (string, error) {
	r := release.NewRelease(releaseImage, pullSecret, c.EnvConfig)
	fileName := fmt.Sprintf(coreOsFileName, cpuArch)
	return r.ExtractFile(machineOsImageName, fileName)
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
	_, err = executer.NewExecuter().Execute(executer.Command{
		Args: strings.Fields(embedCmd),
	})
	return err
}

func (c *coreos) FetchCoreOSStream(releaseImage, pullSecret string) (map[string]any, error) {
	r := release.NewRelease(releaseImage, pullSecret, c.EnvConfig)
	path, err := r.ExtractFile(machineOsImageName, coreOsStream)
	if err != nil {
		return nil, err
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(file, &m); err != nil {
		return nil, errors.Wrap(err, "failed to parse CoreOS stream metadata")
	}

	return m, nil
}
