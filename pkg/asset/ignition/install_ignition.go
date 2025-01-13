package ignition

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"
	"github.com/hashicorp/go-version"
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
	InstallIgnitionPath          = "ignition/install/config.ign"
	baseIgnitionPath             = "ignition/base/config.ign"
	bootDevice                   = "/dev/disk/by-partlabel/boot"
	bootMountPath                = "/boot"
	installRegistryDataPath      = "/mnt/agentdata/oc-mirror/install"
	catalogSourcePattern         = "catalogSource-*.yaml"
	icspFileName                 = "imageContentSourcePolicy.yaml"
	rendezvousHostEnvFilePath    = "/etc/assisted/rendezvous-host.env"
	rendezvousHostEnvPlaceholder = "placeholder-content-for-rendezvous-host.env"
)

var (
	installServices = []string{
		"start-local-registry.service",
		"add-grub-menuitem.service",
		"set-node-zero.service",
	}

	installScripts = []string{
		"setup-local-registry.sh",
		"add-grub-menuitem.sh",
		"set-node-zero.sh",
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
	}
}

// Generate the base ISO.
func (i *InstallIgnition) Generate(_ context.Context, dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(envConfig, applianceConfig)

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

	if swag.BoolValue(applianceConfig.Config.StopLocalRegistry) {
		installServices = append(installServices, "stop-local-registry.service")
		installScripts = append(installScripts, "stop-local-registry.sh")
	}

	// Create install template data
	templateData := templates.GetInstallIgnitionTemplateData(installRegistryDataPath, corePassHash)

	if swag.BoolValue(applianceConfig.Config.CreatePinnedImageSets) && i.isOcpVersionCompatibleWithPinnedImageSet(applianceConfig) {
		installServices = append(installServices, "create-pinned-image-sets.service")
		installScripts = append(installScripts, "create-pinned-image-sets.sh")

		if err := i.addPinnedImageSetConfigFiles(envConfig); err != nil {
			return err
		}
	}

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

	// Add user.cfg file
	if err := i.addRecoveryGrubConfigFile(envConfig.TempDir); err != nil {
		return err
	}

	// Add a placeholder for rendezvous-host.env file
	rendezvousHostEnvFile := ignasset.FileFromString(rendezvousHostEnvFilePath,
		"root", 0644, rendezvousHostEnvPlaceholder)
	i.Config.Storage.Files = append(i.Config.Storage.Files, rendezvousHostEnvFile)

	logrus.Debug("Successfully generated install ignition")

	return nil
}

func (i *InstallIgnition) addPinnedImageSetConfigFiles(envConfig *config.EnvConfig) error {
	mappingFile := envConfig.FindInCache(consts.OcMirrorMappingFileName)
	if mappingFile == "" {
		err := fmt.Errorf("unable to find %s in cache", consts.OcMirrorMappingFileName)
		logrus.Error(err)
		return err
	}
	mappingBytes, err := os.ReadFile(mappingFile)
	if err != nil {
		return err
	}

	var builder strings.Builder
	for _, mapping := range strings.Split(string(mappingBytes), "\n") {
		image := strings.Split(mapping, "=")[0]
		if image != "" {
			builder.WriteString(fmt.Sprintf("  - name: %s\n", image))
		}
	}
	images := builder.String()

	for _, role := range []string{"master", "worker"} {
		outputDir := filepath.Join(envConfig.TempDir, role)
		if err := templates.RenderTemplateFile(
			consts.PinnedImageSetTemplateFile,
			templates.GetPinnedImageSetTemplateData(images, role),
			outputDir); err != nil {
			return err
		}

		fileBytes, err := os.ReadFile(templates.GetFilePathByTemplate(consts.PinnedImageSetTemplateFile, outputDir))
		if err != nil {
			return err
		}

		fileIgnitionConfig := ignasset.FileFromBytes(fmt.Sprintf(consts.PinnedImageSetPattern, role), "root", 0644, fileBytes)
		i.Config.Storage.Files = append(i.Config.Storage.Files, fileIgnitionConfig)
	}

	return nil
}

func (i *InstallIgnition) addRecoveryGrubConfigFile(tempDir string) error {
	// Generate user.cfg
	if err := templates.RenderTemplateFile(
		consts.UserCfgTemplateFile,
		templates.GetUserCfgTemplateData(consts.GrubMenuEntryNameRecovery),
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

func (i *InstallIgnition) isOcpVersionCompatibleWithPinnedImageSet(applianceConfig *config.ApplianceConfig) bool {
	minOcpVer, _ := version.NewVersion(consts.MinOcpVersionForPinnedImageSet)
	ocpVer, _ := version.NewVersion(applianceConfig.Config.OcpRelease.Version)
	return ocpVer.GreaterThanOrEqual(minOcpVer)
}
