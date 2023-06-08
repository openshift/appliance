package installer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/release"
	"github.com/sirupsen/logrus"
)

const (
	templateUnconfiguredIgnitionImage  = "podman run %s agent create unconfigured-ignition --dir %s"
	templateUnconfiguredIgnitionBinary = "%s/openshift-install agent create unconfigured-ignition --dir %s"
	installerImageName                 = "installer"
	unconfiguredIgnitionFileName       = "unconfigured-agent.ign"
)

type Installer interface {
	CreateUnconfiguredIgnition(releaseImage, pullSecret string) (string, error)
}

type installer struct {
	EnvConfig *config.EnvConfig
}

func NewInstaller(envConfig *config.EnvConfig) Installer {
	return &installer{
		EnvConfig: envConfig,
	}
}

func (i *installer) CreateUnconfiguredIgnition(releaseImage, pullSecret string) (string, error) {
	if !i.EnvConfig.DebugBaseIgnition {
		// TODO: remove once the API is ready (see below)
		if true {
			return "pkg/asset/ignition/unconfigured.ign", nil
		}

		// TODO: use logic below once the API is ready ('agent create unconfigured-ignition')
		//       see: https://issues.redhat.com/browse/AGENT-574
		r := release.NewRelease(releaseImage, pullSecret, i.EnvConfig)
		imageUri, err := r.GetImageFromRelease(installerImageName)
		if err != nil {
			return "", err
		}
		createCmd := fmt.Sprintf(templateUnconfiguredIgnitionImage, imageUri, i.EnvConfig.TempDir)
		args := strings.Split(createCmd, " ")
		_, err = executer.NewExecuter().Execute(args[0], args[1:]...)
		return filepath.Join(i.EnvConfig.TempDir, unconfiguredIgnitionFileName), err
	} else {
		logrus.Debugf("Using openshift-install binary from cache dir to fetch unconfigured-ignition")
		cacheDir := filepath.Join(i.EnvConfig.AssetsDir, config.CacheDir)
		createCmd := fmt.Sprintf(templateUnconfiguredIgnitionBinary, cacheDir, i.EnvConfig.TempDir)
		args := strings.Split(createCmd, " ")
		_, err := executer.NewExecuter().Execute(args[0], args[1:]...)
		return filepath.Join(i.EnvConfig.TempDir, unconfiguredIgnitionFileName), err
	}
}
