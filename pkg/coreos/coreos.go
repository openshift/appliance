package coreos

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/danielerez/openshift-appliance/pkg/executer"
	"github.com/danielerez/openshift-appliance/pkg/release"
	"github.com/sirupsen/logrus"
)

const (
	templateDownloadDiskImage = "coreos-installer download -s stable -p qemu -f qcow2.xz --architecture %s --decompress -C %s"
	templateShowISOKargs      = "coreos-installer iso kargs show %s"
	machineOsImageName        = "machine-os-images"
	coreOsFileName            = "coreos/coreos-%s.iso"
)

type CoreOS interface {
	DownloadDiskImage() (string, error)
	DownloadISO(releaseImage, pullSecret string) (string, error)
	GetLiveISOVersion(filePath string) string
}

type coreos struct {
	CacheDir string
}

func NewCoreOS(cacheDir string) *coreos {
	return &coreos{
		CacheDir: cacheDir,
	}
}

func (c *coreos) DownloadDiskImage(cpuArch string) (string, error) {
	downloadCmd := fmt.Sprintf(templateDownloadDiskImage, cpuArch, c.CacheDir)
	args := strings.Split(downloadCmd, " ")
	return executer.NewExecuter().Execute(args[0], args[1:]...)
}

func (c *coreos) DownloadISO(releaseImage, pullSecret string) (string, error) {
	r := release.NewRelease(executer.NewExecuter(), releaseImage, pullSecret, c.CacheDir)
	cpuArch, err := r.GetReleaseArchitecture()
	if err != nil {
		return "", err
	}
	fileName := fmt.Sprintf(coreOsFileName, cpuArch)
	return r.ExtractFile(machineOsImageName, fileName)
}

func (c *coreos) GetLiveISOVersion(filePath string) (string, error) {
	downloadCmd := fmt.Sprintf(templateShowISOKargs, filePath)
	args := strings.Split(downloadCmd, " ")
	kargs, err := executer.NewExecuter().Execute(args[0], args[1:]...)
	if err != nil {
		return "", err
	}
	liveISO := strings.Split(strings.Split(kargs, " ")[0], "=")[1]
	return liveISO, nil
}

func (c *coreos) FindInCache(filePattern string) string {
	files, err := filepath.Glob(filepath.Join(c.CacheDir, filePattern))
	if err != nil {
		logrus.Debugf("Failed searching for file '%s' in dir '%s'", filePattern, c.CacheDir)
		return ""
	}
	if len(files) > 0 {
		return files[0]
	}
	return ""
}
