package ignition

import (
	"os"
	"path/filepath"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	ignitionutil "github.com/openshift/appliance/pkg/ignition"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	ignasset "github.com/openshift/installer/pkg/asset/ignition"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	InstallIgnitionPath     = "ignition/install/config.ign"
	baseIgnitionPath        = "ignition/base/config.ign"
	bootDevice              = "/dev/disk/by-partlabel/boot"
	bootMountPath           = "/boot"
	installRegistryDataPath = "/mnt/agentdata/oc-mirror/install"
	catalogSourcePattern    = "catalogSource-*.yaml"
	icspFileName            = "imageContentSourcePolicy.yaml"
)

var (
	installServices = []string{
		"start-local-registry.service",
	}

	installScripts = []string{
		"setup-local-registry.sh",
	}
)

// InstallIgnition generates the ignition file for cluster installation phase
type InstallIgnition struct {
	Config igntypes.Config
}

var _ asset.Asset = (*InstallIgnition)(nil)

// Name returns the human-friendly name of the asset.
func (i *InstallIgnition) Name() string {
	return "Install ignition"
}

// Dependencies returns dependencies used by the asset.
func (i *InstallIgnition) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
	}
}

// Generate the base ISO.
func (i *InstallIgnition) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(envConfig, applianceConfig)

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


	if swag.BoolValue(applianceConfig.Config.StopLocalRegistry) {
		installServices = append(installServices, "stop-local-registry.service")
		installScripts = append(installScripts, "stop-local-registry.sh")
	}

	// Create install template data
	templateData := templates.GetInstallIgnitionTemplateData(installRegistryDataPath)

	// Add services common for bootstrap and install
	if err := bootstrap.AddSystemdUnits(&i.Config, "services/common", templateData, installServices); err != nil {
		return err
	}

	// Add services exclusive for install
	if err := bootstrap.AddSystemdUnits(&i.Config, "services/install", templateData, installServices); err != nil {
		return err
	}

	// Add install scripts to ignition
	for _, script := range installScripts {
		if err := bootstrap.AddStorageFiles(&i.Config,
			"/usr/local/bin/"+script,
			"scripts/bin/"+script+".template",
			templateData); err != nil {
			return err
		}
	}

	// Add registry.env file
	registryEnvFile := ignasset.FileFromString(consts.RegistryEnvPath,
		"root", 0644, templates.GetRegistryEnv(consts.RegistryDataInstall))
	i.Config.Storage.Files = append(i.Config.Storage.Files, registryEnvFile)

	// Add grub menu item
	if err := i.addRecoveryGrubMenuItem(envConfig.TempDir); err != nil {
		return err
	}

	logrus.Debug("Successfully generated install ignition")

	return nil
}

func (i *InstallIgnition) addRecoveryGrubMenuItem(tempDir string) error {
	if err := templates.RenderTemplateFile(
		consts.UserCfgTemplateFile,
		templates.GetUserCfgTemplateData(consts.GrubMenuEntryNameRecovery, consts.GrubDefaultRecovery),
		tempDir); err != nil {
		return err
	}
	cfgFilePath := templates.GetFilePathByTemplate(consts.UserCfgTemplateFile, tempDir)
	cfgFileBytes, err := os.ReadFile(cfgFilePath)
	if err != nil {
		return err
	}
	cfgFile := ignasset.FileFromBytes(consts.UserCfgFilePath,
		"root", 0644, cfgFileBytes)
	i.Config.Storage.Files = append(i.Config.Storage.Files, cfgFile)
	format := "ext4"
	path := bootMountPath
	i.Config.Storage.Filesystems = append(i.Config.Storage.Filesystems, igntypes.Filesystem{
		Device: bootDevice,
		Format: &format,
		Path:   &path,
	})

	return nil
}

func (i *InstallIgnition) PersistToFile(directory string) error {
	ignition := ignitionutil.NewIgnition(ignitionutil.IgnitionConfig{})

	// Merge with base ignition if exists
	baseConfigPath := filepath.Join(directory, baseIgnitionPath)
	baseConfig, err := ignition.ParseIgnitionFile(baseConfigPath)
	config := &i.Config
	if err == nil {
		config, err = ignition.MergeIgnitionConfig(baseConfig, config)
		if err != nil {
			return err
		}
		logrus.Debugf("Merged install ignition with: %s", baseIgnitionPath)
	}

	configPath := filepath.Join(directory, InstallIgnitionPath)
	if err = os.MkdirAll(filepath.Dir(configPath), os.ModePerm); err != nil {
		return err
	}
	return ignition.WriteIgnitionFile(configPath, config)
}
