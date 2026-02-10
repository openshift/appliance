package ignition

import (
	"os"
	"path/filepath"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/manifests"
	"github.com/openshift/appliance/pkg/consts"
	ignitionutil "github.com/openshift/appliance/pkg/ignition"
	"github.com/openshift/appliance/pkg/registry"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	ignasset "github.com/openshift/installer/pkg/asset/ignition"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	InstallIgnitionPath          = "ignition/install/config.ign"
	baseIgnitionPath             = "ignition/base/config.ign"
	bootDevice                   = "/dev/disk/by-partlabel/boot"
	bootMountPath                = "/boot"
	rendezvousHostEnvFilePath    = "/etc/assisted/rendezvous-host.env"
	rendezvousHostEnvPlaceholder = "placeholder-content-for-rendezvous-host.env"
	postInstallationCrsDir       = "post-installation"
)

var (
	installServices = []string{
		"set-node-zero.service",
		"apply-operator-crs.service",
		"watch-iri-tls-certs.path",
	}

	installScripts = []string{
		"set-node-zero.sh",
		"setup-local-registry-upgrade.sh",
		"start-cluster-upgrade.sh",
		"stop-local-registry.sh",
		"mount-agent-data.sh",
		"apply-operator-crs.sh",
		"reconfigure-local-registry-iri-tls.sh",
	}

	corePassHash string
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
		&manifests.OperatorCRs{},
	}
}

// Generate the base ISO.
func (i *InstallIgnition) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	operatorCRs := &manifests.OperatorCRs{}
	dependencies.Get(envConfig, applianceConfig, operatorCRs)

	// Determine if we're using the OCP registry (for the podman run command)
	useOcpRegistry := registry.ShouldUseOcpRegistry(envConfig, applianceConfig)
	if useOcpRegistry {
		logrus.Debug("InstallIgnition will use OCP docker-registry image")
	}

	i.Config = igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: igntypes.MaxVersion.String(),
		},
	}

	if applianceConfig.Config.UserCorePass != nil {
		// Generate core pass hash
		passBytes, err := bcrypt.GenerateFromPassword([]byte(*applianceConfig.Config.UserCorePass), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		corePassHash = string(passBytes)
	}

	if !swag.BoolValue(applianceConfig.Config.SkipLocalRegistry) {
		installServices = append(installServices, "start-local-registry.service")
		installScripts = append(installScripts, "load-registry-image.sh", "setup-local-registry.sh")
	}

	if swag.BoolValue(applianceConfig.Config.StopLocalRegistry) {
		installServices = append(installServices, "stop-local-registry.service")
	}

	if swag.BoolValue(applianceConfig.Config.CreatePinnedImageSets) {
		installServices = append(installServices, "create-pinned-image-sets.service")
		installScripts = append(installScripts, "create-pinned-image-sets.sh")
	}

	if !envConfig.IsLiveISO {
		installServices = append(installServices, "add-grub-menuitem.service")
		installScripts = append(installScripts, "add-grub-menuitem.sh")

		// Add user.cfg file
		if err := i.addRecoveryGrubConfigFile(envConfig.TempDir, applianceConfig.Config.EnableFips); err != nil {
			return err
		}
	}

	// Create install template data
	templateData := templates.GetInstallIgnitionTemplateData(
		envConfig.IsLiveISO,
		swag.BoolValue(applianceConfig.Config.EnableInteractiveFlow),
		corePassHash,
	)

	// Add registry service from appropriate directory (OCP or default)
	registryServiceDir := "services/local-registry-default"
	if useOcpRegistry {
		registryServiceDir = "services/local-registry-ocp"
	}
	if err := bootstrap.AddSystemdUnits(&i.Config, registryServiceDir, templateData, installServices); err != nil {
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

	// Add udev file
	err := bootstrap.AddStorageFiles(&i.Config, "/etc/udev", "udev", nil)
	if err != nil {
		return err
	}

	// Add registry.env file
	registryImageURI := registry.GetRegistryImageURI(envConfig, applianceConfig)
	registryEnvFile := ignasset.FileFromString(consts.RegistryEnvPath,
		"root", 0644, templates.GetRegistryEnv(registryImageURI, consts.RegistryDataInstall, consts.RegistryDataUpgrade))
	i.Config.Storage.Files = append(i.Config.Storage.Files, registryEnvFile)

	// Add a placeholder for rendezvous-host.env file
	rendezvousHostEnvFile := ignasset.FileFromString(rendezvousHostEnvFilePath,
		"root", 0644, rendezvousHostEnvPlaceholder)
	i.Config.Storage.Files = append(i.Config.Storage.Files, rendezvousHostEnvFile)

	// Add operators CR manifests from 'openshift/crs' dir
	if err := addExtraManifests(
		&i.Config,
		operatorCRs.FileList,
		filepath.Join(extraManifestsPath, postInstallationCrsDir),
		swag.Bool(false)); err != nil {
		return err
	}

	logrus.Debug("Successfully generated install ignition")

	return nil
}

func (i *InstallIgnition) addRecoveryGrubConfigFile(tempDir string, enableFips *bool) error {
	// Generate user.cfg
	if err := templates.RenderTemplateFile(
		consts.UserCfgTemplateFile,
		templates.GetUserCfgTemplateData(consts.GrubMenuEntryNameRecovery, swag.BoolValue(enableFips)),
		tempDir); err != nil {
		return err
	}
	cfgFilePath := templates.GetFilePathByTemplate(consts.UserCfgTemplateFile, tempDir)
	cfgFileBytes, err := os.ReadFile(cfgFilePath)
	if err != nil {
		return err
	}
	cfgFile := ignasset.FileFromBytes(consts.UserCfgFilePath, "root", 0644, cfgFileBytes)

	// Append user.cfg to Files
	i.Config.Storage.Files = append(i.Config.Storage.Files, cfgFile)

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
