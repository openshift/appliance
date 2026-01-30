package ignition

import (
	"os"
	"path/filepath"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/openshift/installer/pkg/asset/ignition"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"golang.org/x/crypto/bcrypt"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
)

var (
	deployServices = []string{
		"deploy.service",
	}

	deployScripts = []string{
		"deploy.sh",
	}
)

type DeployIgnition struct {
	Config igntypes.Config
}

var _ asset.Asset = (*BootstrapIgnition)(nil)

// Name returns the human-friendly name of the asset.
func (i *DeployIgnition) Name() string {
	return "Deploy ignition"
}

// Dependencies returns dependencies used by the asset.
func (i *DeployIgnition) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
		&config.DeployConfig{},
	}
}

// Generate the base ISO.
func (i *DeployIgnition) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	deployConfig := &config.DeployConfig{}
	dependencies.Get(envConfig, applianceConfig, deployConfig)

	i.Config = igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: igntypes.MaxVersion.String(),
		},
	}

	passwdUser := igntypes.PasswdUser{
		Name: "core",
	}

	if applianceConfig.Config.UserCorePass != nil {
		// Add user 'core' password
		passBytes, err := bcrypt.GenerateFromPassword([]byte(*applianceConfig.Config.UserCorePass), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		pwdHash := string(passBytes)
		passwdUser.PasswordHash = &pwdHash
	}

	// Add public ssh key
	if applianceConfig.Config.SshKey != nil {
		passwdUser.SSHAuthorizedKeys = []igntypes.SSHAuthorizedKey{
			igntypes.SSHAuthorizedKey(*applianceConfig.Config.SshKey),
		}
	}
	i.Config.Passwd.Users = append(i.Config.Passwd.Users, passwdUser)

	// Create template data
	templateData := templates.GetDeployIgnitionTemplateData(
		deployConfig.TargetDevice, deployConfig.PostScript, deployConfig.SparseClone, deployConfig.DryRun)

	// Add deploy services
	if err := bootstrap.AddSystemdUnits(&i.Config, "services/deploy", templateData, deployServices); err != nil {
		return err
	}

	// Add deploy scripts to ignition
	for _, script := range deployScripts {
		if err := bootstrap.AddStorageFiles(&i.Config,
			filepath.Join("/usr/local/bin/", script),
			"scripts/bin/"+script+".template",
			templateData); err != nil {
			return err
		}
	}

	// Add post script if specified
	if deployConfig.PostScript != "" {
		data, err := os.ReadFile(filepath.Join(envConfig.AssetsDir, deployConfig.PostScript))
		if err != nil {
			return err
		}
		postScript := filepath.Join("/usr/local/bin/", deployConfig.PostScript)
		file := ignition.FileFromBytes(postScript, "root", 0755, data)
		i.Config.Storage.Files = append(i.Config.Storage.Files, file)
	}

	return nil
}
