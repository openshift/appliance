package coreos

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/danielerez/openshift-appliance/pkg/release"
	"github.com/itchyny/gojq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	templateDownloadDiskImage = "coreos-installer download -s stable -p qemu -f qcow2.xz --architecture %s --decompress -C %s"
	templateEmbedIgnition     = "coreos-installer iso ignition embed -f --ignition-file %s %s"
	machineOsImageName        = "machine-os-images"
	coreOsFileName            = "coreos/coreos-%s.iso"
	coreOsStream              = "coreos/coreos-stream.json"
	coreOsDiskImageUrlQuery   = ".architectures.x86_64.artifacts.qemu.formats[\"qcow2.gz\"].disk.location"
	coreOsDiskImageGz         = "coreos.tar.gz"
)

type CoreOS interface {
	DownloadDiskImage(releaseImage, pullSecret string) (string, error)
	DownloadISO(releaseImage, pullSecret string) (string, error)
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
	compressed := filepath.Join(c.EnvConfig.TempDir, coreOsDiskImageGz)
	c.downloadFile(compressed, qcowGzUrl)
	if err != nil {
		return "", err
	}

	return c.ungzip(compressed, c.EnvConfig.CacheDir)
}

func (c *coreos) DownloadISO(releaseImage, pullSecret string) (string, error) {
	r := release.NewRelease(releaseImage, pullSecret, c.EnvConfig)
	cpuArch, err := r.GetReleaseArchitecture()
	if err != nil {
		return "", err
	}
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
	args := strings.Split(embedCmd, " ")
	_, err = executer.NewExecuter().Execute(args[0], args[1:]...)
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

func (c *coreos) downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create output file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (c *coreos) ungzip(source, target string) (string, error) {
	reader, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return "", err
	}
	defer archive.Close()

	target = filepath.Join(target, archive.Name)
	writer, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return target, err
}
