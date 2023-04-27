package coreos

import (
	"fmt"
	"os"
	"strings"

	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/danielerez/openshift-appliance/pkg/release"
	"github.com/sirupsen/logrus"
)

const (
	templateDownloadDiskImage = "coreos-installer download -s stable -p qemu -f qcow2.xz --architecture %s --decompress -C %s"
	templateEmbedIgnition     = "coreos-installer iso ignition embed -f --ignition-file %s %s"
	machineOsImageName        = "machine-os-images"
	coreOsFileName            = "coreos/coreos-%s.iso"
)

type CoreOS interface {
	DownloadDiskImage() (string, error)
	DownloadISO(releaseImage, pullSecret string) (string, error)
	EmbedIgnition(ignition []byte, isoPath string) error
}

type coreos struct {
	CacheDir  string
	AssetsDir string
	TempDir   string
}

func NewCoreOS(envConfig *config.EnvConfig) *coreos {
	return &coreos{
		CacheDir:  envConfig.CacheDir,
		AssetsDir: envConfig.AssetsDir,
		TempDir:   envConfig.TempDir,
	}
}

func (c *coreos) DownloadDiskImage(cpuArch string) (string, error) {
	downloadCmd := fmt.Sprintf(templateDownloadDiskImage, cpuArch, c.CacheDir)
	args := strings.Split(downloadCmd, " ")
	return executer.NewExecuter().Execute(args[0], args[1:]...)
}

func (c *coreos) DownloadISO(releaseImage, pullSecret string) (string, error) {
	envConfig := &config.EnvConfig{CacheDir: c.CacheDir, AssetsDir: c.AssetsDir, TempDir: c.TempDir}
	r := release.NewRelease(releaseImage, pullSecret, envConfig)
	cpuArch, err := r.GetReleaseArchitecture()
	if err != nil {
		return "", err
	}
	fileName := fmt.Sprintf(coreOsFileName, cpuArch)
	return r.ExtractFile(machineOsImageName, fileName)
}

func (c *coreos) EmbedIgnition(ignition []byte, isoPath string) error {
	// Write ignition to a temporary file
	ignitionFile, err := os.CreateTemp(c.TempDir, "config.ign")
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
