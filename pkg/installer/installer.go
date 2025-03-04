package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-openapi/swag"

	"github.com/hashicorp/go-version"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/log"
	"github.com/openshift/appliance/pkg/release"
	"github.com/sirupsen/logrus"
)

const (
	installerBinaryName                = "openshift-install"
	installerFipsBinaryName            = "openshift-install-fips"
	installerBinaryGZ                  = "openshift-install-linux.tar.gz"
	templateUnconfiguredIgnitionBinary = "%s agent create unconfigured-ignition --dir %s"
	templateInstallerDownloadURL       = "https://mirror.openshift.com/pub/openshift-v%s/%s/clients/%s/%s/openshift-install-linux.tar.gz"
	unconfiguredIgnitionFileName       = "unconfigured-agent.ign"
)

type Installer interface {
	CreateUnconfiguredIgnition() (string, error)
	GetInstallerDownloadURL() (string, error)
	GetInstallerBinaryName() string
}

type InstallerConfig struct {
	Executer            executer.Executer
	EnvConfig           *config.EnvConfig
	Release             release.Release
	ApplianceConfig     *config.ApplianceConfig
	InstallerBinaryName string
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

	inst := &installer{
		InstallerConfig: config,
	}
	inst.InstallerBinaryName = inst.GetInstallerBinaryName()

	return inst
}

func (i *installer) CreateUnconfiguredIgnition() (string, error) {
	var openshiftInstallFilePath string
	var err error

	if !i.EnvConfig.DebugBaseIgnition {
		if fileName := i.EnvConfig.FindInCache(i.InstallerBinaryName); fileName != "" {
			logrus.Infof("Reusing %s binary from cache", i.InstallerBinaryName)
			openshiftInstallFilePath = fileName
		} else {
			openshiftInstallFilePath, err = i.downloadInstallerBinary()
			if err != nil {
				return "", err
			}
		}
	} else {
		logrus.Debugf("Using openshift-install binary from assets dir to fetch unconfigured-ignition")
		openshiftInstallFilePath = filepath.Join(i.EnvConfig.AssetsDir, i.InstallerBinaryName)
	}

	createCmd := fmt.Sprintf(templateUnconfiguredIgnitionBinary, openshiftInstallFilePath, i.EnvConfig.TempDir)
	if swag.BoolValue(i.ApplianceConfig.Config.EnableInteractiveFlow) {
		createCmd = fmt.Sprintf("%s --interactive", createCmd)
	}
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

	ocpClient := "ocp"
	if strings.Contains(releaseVersion.String(), "-ec") {
		ocpClient = "ocp-dev-preview"
	}

	return fmt.Sprintf(templateInstallerDownloadURL, majorVersion, cpuArch, ocpClient, releaseVersion), nil
}

func (i *installer) downloadInstallerBinary() (string, error) {
	spinner := log.NewSpinner(
		fmt.Sprintf("Fetching %s binary...", i.InstallerBinaryName),
		fmt.Sprintf("Successfully fetched %s binary", i.InstallerBinaryName),
		fmt.Sprintf("Failed to fetch %s binary", i.InstallerBinaryName),
		i.EnvConfig,
	)

	logrus.Debugf("Fetch %s binary from release payload", i.InstallerBinaryName)
	stdout, err := i.Release.ExtractCommand(i.InstallerBinaryName, i.EnvConfig.CacheDir)
	if err != nil {
		logrus.Errorf("%s", stdout)
		return "", log.StopSpinner(spinner, err)
	}

	err = log.StopSpinner(spinner, nil)
	if err != nil {
		return "", err
	}

	installerBinaryPath := filepath.Join(i.EnvConfig.CacheDir, i.InstallerBinaryName)
	err = os.Chmod(installerBinaryPath, 0755)
	if err != nil {
		// return "", err
		logrus.Warnf("%s", err)
	}
	return installerBinaryPath, nil
}

func (i *installer) GetInstallerBinaryName() string {
	if swag.BoolValue(i.ApplianceConfig.Config.EnableFips) {
		return installerFipsBinaryName
	}
	return installerBinaryName
}
