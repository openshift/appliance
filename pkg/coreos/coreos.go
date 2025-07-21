package coreos

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cavaliercoder/go-cpio"
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
	coreOsDiskImageUrlQuery = ".architectures.x86_64.artifacts.metal.formats[\"raw.gz\"].disk.location"

	CoreOsDiskImageGz = "coreos.tar.gz"
)

type CoreOS interface {
	DownloadDiskImage() (string, error)
	DownloadISO() (string, error)
	EmbedIgnition(ignition []byte, isoPath string) error
	WrapIgnition(ignition []byte, ignitionPath, imagePath string) error
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

func (c *coreos) WrapIgnition(ignition []byte, ignitionPath, imagePath string) error {
	ignitionImgFile, err := os.OpenFile(imagePath, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return err
	}
	defer func() {
		ignitionImgFile.Close()
	}()


	compressedCpio, err := generateCompressedCPIO(ignition, ignitionPath, 0o100_644)
	if err != nil {
		return err
	}

	_, err = ignitionImgFile.Write(compressedCpio)
	if err != nil {
		logrus.Errorf("Failed to write ignition data into %s: %s", ignitionImgFile.Name(), err.Error())
		return err
	}

	return nil
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

func generateCompressedCPIO(fileContent []byte, filePath string, mode cpio.FileMode) ([]byte, error) {
	// Run gzip compression
	compressedBuffer := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(compressedBuffer)
	// Create CPIO archive
	cpioWriter := cpio.NewWriter(gzipWriter)

	if err := cpioWriter.WriteHeader(&cpio.Header{
		Name: filePath,
		Mode: mode,
		Size: int64(len(fileContent)),
	}); err != nil {
		return nil, errors.Wrap(err, "Failed to write CPIO header")
	}
	if _, err := cpioWriter.Write(fileContent); err != nil {
		return nil, errors.Wrap(err, "Failed to write CPIO archive")
	}

	if err := cpioWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "Failed to close CPIO archive")
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "Failed to gzip ignition config")
	}

	padSize := (4 - (compressedBuffer.Len() % 4)) % 4
	for i := 0; i < padSize; i++ {
		if err := compressedBuffer.WriteByte(0); err != nil {
			return nil, err
		}
	}

	return compressedBuffer.Bytes(), nil
}
