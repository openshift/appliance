package installer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/release"
	"github.com/sirupsen/logrus"
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
	Release         release.Release
	ApplianceConfig *config.ApplianceConfig
}

type installer struct {
	InstallerConfig
}

func NewInstaller(config InstallerConfig) Installer {
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

	return &installer{
		InstallerConfig: config,
	}
}

func (i *installer) CreateUnconfiguredIgnition() (string, error) {
	var openshiftInstallFilePath string
	var err error

	if !i.EnvConfig.DebugBaseIgnition {
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

	logrus.Debugf("Fetch openshift-install binary from release payload %s", *i.ApplianceConfig.Config.OcpRelease.URL)
	stdout, err := i.Release.ExtractCommand(installerBinaryName, i.EnvConfig.CacheDir)
	if err != nil {
		logrus.Errorf("%s", stdout)
		return "", log.StopSpinner(spinner, err)
	}

	err = log.StopSpinner(spinner, nil)
	if err != nil {
		return "", err
	}

	installerBinaryPath := filepath.Join(i.EnvConfig.CacheDir, installerBinaryName)
	err = os.Chmod(installerBinaryPath, 0755)
	if err != nil {
		return "", err
	}
	return installerBinaryPath, nil
}
