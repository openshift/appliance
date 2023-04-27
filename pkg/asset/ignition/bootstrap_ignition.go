package ignition

import (
	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/danielerez/openshift-appliance/pkg/asset/config"
	"github.com/danielerez/openshift-appliance/pkg/asset/registry"
	"github.com/danielerez/openshift-appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	assetignition "github.com/openshift/installer/pkg/asset/ignition"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/password"
	"github.com/sirupsen/logrus"
)

var (
	bootstrapServices = []string{
		"start-local-registry.service",
	}

	bootstrapScripts = []string{
		"start-local-registry.sh",
		"get-container-images.sh",
		"extract-agent.sh",
	}
)

const (
	bootstrapRegistryDataPath = "/mnt/agentdata/oc-mirror/bootstrap"
	registriesConfFilePath    = "/etc/containers/registries.conf"
)

// BootstrapIgnition generates the bootstrap ignition file for the recovery ISO
type BootstrapIgnition struct {
	Config igntypes.Config
}

var _ asset.Asset = (*BootstrapIgnition)(nil)

// Name returns the human-friendly name of the asset.
func (i *BootstrapIgnition) Name() string {
	return "Bootstrap Ignition"
}

// Dependencies returns dependencies used by the asset.
func (i *BootstrapIgnition) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.ApplianceConfig{},
		&password.KubeadminPassword{},
		&registry.RegistriesConf{},
	}
}

// Generate the base ISO.
func (i *BootstrapIgnition) Generate(dependencies asset.Parents) error {
	applianceConfig := &config.ApplianceConfig{}
	pwd := &password.KubeadminPassword{}
	registryConf := &registry.RegistriesConf{}
	dependencies.Get(applianceConfig, pwd, registryConf)

	i.Config = igntypes.Config{}

	// Add bootstrap services to ignition
	if err := bootstrap.AddSystemdUnits(&i.Config, "services", nil, bootstrapServices); err != nil {
		return err
	}

	// Add bootstrap scripts to ignition
	templateData := templates.GetBootstrapIgnitionTemplateData(
		applianceConfig.Config.OcpRelease, bootstrapRegistryDataPath)
	for _, script := range bootstrapScripts {
		if err := bootstrap.AddStorageFiles(&i.Config,
			"/usr/local/bin/"+script,
			"scripts/bin/"+script+".template",
			templateData); err != nil {
			return err
		}
	}

	// Add public ssh key
	pwdHash := string(pwd.PasswordHash)
	passwdUser := igntypes.PasswdUser{
		Name:         "core",
		PasswordHash: &pwdHash,
	}
	if applianceConfig.Config.SshKey != nil {
		passwdUser.SSHAuthorizedKeys = []igntypes.SSHAuthorizedKey{
			igntypes.SSHAuthorizedKey(*applianceConfig.Config.SshKey),
		}
	}
	i.Config.Passwd.Users = append(i.Config.Passwd.Users, passwdUser)

	registriesFile := assetignition.FileFromBytes(registriesConfFilePath,
		"root", 0600, registryConf.FileData)
	i.Config.Storage.Files = append(i.Config.Storage.Files, registriesFile)

	logrus.Debug("Successfully generated bootstrap ignition")

	return nil
}
