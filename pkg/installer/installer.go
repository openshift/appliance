package installer

import (
	"fmt"
	"path/filepath"

	"github.com/cavaliergopher/grab/v3"
	"github.com/hashicorp/go-version"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/walle/targz"
)

const (
	installerBinaryName                = "openshift-install"
	installerBinaryGZ                  = "openshift-install-linux.tar.gz"
	templateUnconfiguredIgnitionBinary = "%s agent create unconfigured-ignition --dir %s"
	templateInstallerDownloadURL       = "https://mirror.openshift.com/pub/openshift-v%s/%s/clients/ocp/%s/openshift-install-linux.tar.gz"
	unconfiguredIgnitionFileName       = "unconfigured-agent.ign"
)

type Installer interface {
	CreateUnconfiguredIgnition() (string, error)
	GetInstallerDownloadURL() (string, error)
}

type InstallerConfig struct {
	Executer        executer.Executer
	EnvConfig       *config.EnvConfig
	ApplianceConfig *config.ApplianceConfig
}

type installer struct {
	InstallerConfig
}

func NewInstaller(config InstallerConfig) Installer {
	if config.Executer == nil {
		config.Executer = executer.NewExecuter()
	}

	return &installer{
		InstallerConfig: config,
	}
}

func (i *installer) CreateUnconfiguredIgnition() (string, error) {
	var openshiftInstallFilePath string
	var err error

	if !i.EnvConfig.DebugBaseIgnition {
		// TODO: remove once the API is ready (see below)
		if true {
			return "pkg/asset/ignition/unconfigured.ign", nil
		}

		// TODO: use logic below once the API is ready ('agent create unconfigured-ignition')
		//       see: https://issues.redhat.com/browse/AGENT-574
		if fileName := i.EnvConfig.FindInCache(installerBinaryName); fileName != "" {
			logrus.Info("Reusing openshift-install binary from cache")
			openshiftInstallFilePath = fileName
		} else {
			openshiftInstallFilePath, err = i.downloadInstallerBinary()
			if err != nil {
				return "", err
			}
		}
	} else {
		logrus.Debugf("Using openshift-install binary from assets dir to fetch unconfigured-ignition")
		openshiftInstallFilePath = filepath.Join(i.EnvConfig.AssetsDir, installerBinaryName)
	}

	createCmd := fmt.Sprintf(templateUnconfiguredIgnitionBinary, openshiftInstallFilePath, i.EnvConfig.TempDir)
	_, err = i.Executer.Execute(createCmd)
	return filepath.Join(i.EnvConfig.TempDir, unconfiguredIgnitionFileName), err
}

func (i *installer) GetInstallerDownloadURL() (string, error) {
	releaseVersion, err := version.NewVersion(i.ApplianceConfig.Config.OcpRelease.Version)
	if err != nil {
		return "", err
	}
	majorVersion := fmt.Sprint(releaseVersion.Segments()[0])
	cpuArch := i.ApplianceConfig.GetCpuArchitecture()

	return fmt.Sprintf(templateInstallerDownloadURL, majorVersion, cpuArch, releaseVersion), nil
}

func (i *installer) downloadInstallerBinary() (string, error) {
	spinner := log.NewSpinner(
		"Fetching openshift-install binary...",
		"Successfully fetched openshift-install binary",
		"Failed to fetch openshift-install binary",
		i.EnvConfig,
	)

	logrus.Debugf("Fetch openshift-install binary from mirror.openshift.com")
	installerDownloadURL, err := i.GetInstallerDownloadURL()
	if err != nil {
		return "", log.StopSpinner(spinner, err)
	}
	compressed := filepath.Join(i.EnvConfig.TempDir, installerBinaryGZ)
	_, err = grab.Get(compressed, installerDownloadURL)
	if err != nil {
		return "", log.StopSpinner(spinner, err)
	}
	if err = targz.Extract(compressed, i.EnvConfig.CacheDir); err != nil {
		return "", log.StopSpinner(spinner, err)
	}
	err = log.StopSpinner(spinner, nil)
	if err != nil {
		return "", err
	}
	return filepath.Join(i.EnvConfig.CacheDir, installerBinaryName), nil
}
