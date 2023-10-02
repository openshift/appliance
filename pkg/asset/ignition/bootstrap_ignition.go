package ignition

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/go-openapi/swag"
	"github.com/pkg/errors"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	agentManifests "github.com/openshift/installer/pkg/asset/agent/manifests"
	ignasset "github.com/openshift/installer/pkg/asset/ignition"
	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/yaml"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/asset/manifests"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/fileutil"
	"github.com/openshift/appliance/pkg/templates"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/asset/password"
	mcfgv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	bootstrapRegistryDataPath = "/mnt/agentdata/oc-mirror/bootstrap"
	registriesConfFilePath    = "/etc/containers/registries.conf"
	manifestPath              = "/etc/assisted/manifests"
	corePassOverrideFilePath  = "/etc/assisted/appliance-override-password-set"
	extraManifestPath         = "/etc/assisted/extra-manifests"
)

var (
	bootstrapServices = []string{
		"start-local-registry.service",
		"assisted-service.service",
		"create-cluster-and-infraenv.service",
		"pre-install.service",
		"pre-install-node-zero.service",
		"update-hosts.service",
	}

	bootstrapScripts = []string{
		"setup-local-registry.sh",
		"set-env-files.sh",
		"pre-install.sh",
		"pre-install-node-zero.sh",
		"release-image-download.sh",
		"release-image.sh",
		"update-hosts.sh",
		"create-virtual-device.sh",
	}
)

// BootstrapIgnition generates the bootstrap ignition file for the recovery ISO
type BootstrapIgnition struct {
	Config igntypes.Config
}

var _ asset.Asset = (*BootstrapIgnition)(nil)

// Name returns the human-friendly name of the asset.
func (i *BootstrapIgnition) Name() string {
	return "Bootstrap ignition"
}

// Dependencies returns dependencies used by the asset.
func (i *BootstrapIgnition) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.EnvConfig{},
		&config.ApplianceConfig{},
		&password.KubeadminPassword{},
		&manifests.ClusterImageSet{},
		&agentManifests.ExtraManifests{},
		&InstallIgnition{},
	}
}

// Generate the base ISO.
func (i *BootstrapIgnition) Generate(dependencies asset.Parents) error {
	envConfig := &config.EnvConfig{}
	applianceConfig := &config.ApplianceConfig{}
	extraManifests := &agentManifests.ExtraManifests{}
	installIgnition := &InstallIgnition{}
	dependencies.Get(envConfig, applianceConfig, extraManifests, installIgnition)

	i.Config = igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: igntypes.MaxVersion.String(),
		},
	}

	if envConfig.DebugBootstrap {
		// Avoid machine reboot after bootstrap to debug install ignition
		bootstrapServices = append(bootstrapServices, "ironic-agent.service")
	}

	// Add services common for bootstrap and install
	if err := bootstrap.AddSystemdUnits(&i.Config, "services/common", nil, bootstrapServices); err != nil {
		return err
	}

	// Add services exclusive for bootstrap
	if err := bootstrap.AddSystemdUnits(&i.Config, "services/bootstrap", nil, bootstrapServices); err != nil {
		return err
	}

	// Fetch install ignition config
	installIgnitionConfig, err := json.Marshal(installIgnition.Config)
	if err != nil {
		return err
	}

	// Get base image path
	coreosImagePattern := fmt.Sprintf(consts.CoreosImagePattern, applianceConfig.GetCpuArchitecture())
	coreosImagePath := envConfig.FindInCache(coreosImagePattern)

	// Add bootstrap scripts to ignition
	templateData := templates.GetBootstrapIgnitionTemplateData(
		applianceConfig.Config.OcpRelease,
		bootstrapRegistryDataPath,
		string(installIgnitionConfig),
		coreosImagePath)
	for _, script := range bootstrapScripts {
		if err := bootstrap.AddStorageFiles(&i.Config,
			"/usr/local/bin/"+script,
			"scripts/bin/"+script+".template",
			templateData); err != nil {
			return err
		}
	}

	passwdUser := igntypes.PasswdUser{
		Name: "core",
	}
	// Add user 'core' password
	if applianceConfig.Config.UserCorePass != nil {
		passBytes, err := bcrypt.GenerateFromPassword([]byte(*applianceConfig.Config.UserCorePass), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		pwdHash := string(passBytes)
		passwdUser.PasswordHash = &pwdHash

		// Add 'appliance-override-password-set' file
		// (needed as an indication that the appliance override the core pass)
		overridePassFile := ignasset.FileFromString(
			corePassOverrideFilePath, "root", 0644, "")
		i.Config.Storage.Files = append(i.Config.Storage.Files, overridePassFile)

		// Add MachineConfigs to set core password post installation
		if err = i.setCoreUserPass("master", pwdHash); err != nil {
			return err
		}
		if err = i.setCoreUserPass("worker", pwdHash); err != nil {
			return err
		}
	}

	// Add registry.env file
	registryEnvFile := ignasset.FileFromString(consts.RegistryEnvPath,
		"root", 0644, templates.GetRegistryEnv(consts.RegistryDataBootstrap))
	i.Config.Storage.Files = append(i.Config.Storage.Files, registryEnvFile)

	// Add public ssh key
	if applianceConfig.Config.SshKey != nil {
		passwdUser.SSHAuthorizedKeys = []igntypes.SSHAuthorizedKey{
			igntypes.SSHAuthorizedKey(*applianceConfig.Config.SshKey),
		}
	}
	i.Config.Passwd.Users = append(i.Config.Passwd.Users, passwdUser)

	if err = i.addExtraManifests(&i.Config, extraManifests); err != nil {
		return err
	}

	// Add ImageContentSourcePolicy manifests generated by oc-mirror
	if err := i.addICSPManifests(envConfig); err != nil {
		return err
	}

	// Add CatalogSource manifests generated by oc-mirror
	if applianceConfig.Config.Operators != nil {
		if err := i.addCatalogSourceManifests(envConfig); err != nil {
			return err
		}
	}

	// Disable all default CatalogSources to avoid failure on disconnected envs
	if !swag.BoolValue(applianceConfig.Config.EnableDefaultSources) {
		if err := i.disableDefaultCatalogSources(); err != nil {
			return err
		}
	}

	logrus.Debug("Successfully generated bootstrap ignition")

	return nil
}

// addExtraManifests is a non-exportable function copy-over from openshift/installer/pkg/asset/agent/image/ignition.go
func (i *BootstrapIgnition) addExtraManifests(config *igntypes.Config, extraManifests *agentManifests.ExtraManifests) error {
	user := "root"
	mode := 0644

	config.Storage.Directories = append(config.Storage.Directories, igntypes.Directory{
		Node: igntypes.Node{
			Path: extraManifestPath,
			User: igntypes.NodeUser{
				Name: &user,
			},
			Overwrite: util.BoolToPtr(true),
		},
		DirectoryEmbedded1: igntypes.DirectoryEmbedded1{
			Mode: &mode,
		},
	})

	for _, file := range extraManifests.FileList {

		type unstructured map[string]interface{}

		yamlList, err := agentManifests.GetMultipleYamls[unstructured](file.Data)
		if err != nil {
			return errors.Wrapf(err, "could not decode YAML for %s", file.Filename)
		}

		for n, manifest := range yamlList {
			m, err := yaml.Marshal(manifest)
			if err != nil {
				return err
			}

			base := filepath.Base(file.Filename)
			ext := filepath.Ext(file.Filename)
			baseWithoutExt := strings.TrimSuffix(base, ext)
			baseFileName := filepath.Join(extraManifestPath, baseWithoutExt)
			fileName := fmt.Sprintf("%s-%d%s", baseFileName, n, ext)

			extraFile := ignasset.FileFromBytes(fileName, user, mode, m)
			config.Storage.Files = append(config.Storage.Files, extraFile)
		}
	}

	return nil
}

func (i *BootstrapIgnition) addICSPManifests(envConfig *config.EnvConfig) error {
	osInterface := &fileutil.OSFS{}

	// Read ImageContentSourcePolicy manifests
	icspListBytes, err := osInterface.ReadFile(filepath.Join(envConfig.CacheDir, icspFileName))
	if err != nil {
		logrus.Error("Missing imageContentSourcePolicy.yaml file in cache")
		return err
	}

	// Split ICSP yaml file and add to extra-manifests
	icspList := strings.Split(string(icspListBytes), "---")
	for c, icsp := range icspList {
		if icsp == "" {
			continue
		}
		icspManifestPath := fmt.Sprintf("%s/icsp-%d.yaml", extraManifestsPath, c)
		icspFile := ignasset.FileFromBytes(icspManifestPath, "root", 0644, []byte(icsp))
		i.Config.Storage.Files = append(i.Config.Storage.Files, icspFile)
	}

	return nil
}

func (i *BootstrapIgnition) addCatalogSourceManifests(envConfig *config.EnvConfig) error {
	// Find CatalogSource manifests
	csFiles, err := filepath.Glob(filepath.Join(envConfig.CacheDir, catalogSourcePattern))
	if err != nil {
		logrus.Error("Missing 'CatalogSource' yaml files in cache")
		return err
	}

	// Read each manifest and add to extra-manifests
	osInterface := &fileutil.OSFS{}
	for _, csFile := range csFiles {
		csBytes, err := osInterface.ReadFile(csFile)
		if err != nil {
			return err
		}
		csManifestPath := fmt.Sprintf("%s/%s", extraManifestsPath, filepath.Base(csFile))
		csFile := ignasset.FileFromBytes(csManifestPath, "root", 0644, csBytes)
		i.Config.Storage.Files = append(i.Config.Storage.Files, csFile)
	}

	return nil
}

func (i *BootstrapIgnition) disableDefaultCatalogSources() error {
	operatorHub := configv1.OperatorHub{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.openshift.io/v1",
			Kind:       "OperatorHub",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.OperatorHubSpec{
			DisableAllDefaultSources: true,
		},
	}
	manifestBytes, err := yaml.Marshal(operatorHub)
	if err != nil {
		return err
	}
	manifestPath := fmt.Sprintf("%s/operatorhub-%s.yaml", extraManifestsPath, operatorHub.Name)
	manifestFile := ignasset.FileFromBytes(manifestPath, "root", 0644, manifestBytes)
	i.Config.Storage.Files = append(i.Config.Storage.Files, manifestFile)

	return nil
}

func (i *BootstrapIgnition) setCoreUserPass(role, pwdHash string) error {
	// Generate ignition config with user core pass
	ignConfig := igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: igntypes.MaxVersion.String(),
		},
		Passwd: igntypes.Passwd{
			Users: []igntypes.PasswdUser{{
				Name: "core", PasswordHash: &pwdHash,
			}},
		},
	}
	ignitionRawExt, err := ignasset.ConvertToRawExtension(ignConfig)
	if err != nil {
		return err
	}

	// Generate the MachineConfig with ignition config
	machineConfig := &mcfgv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcfgv1.SchemeGroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("99-%s-set-core-pass", role),
			Labels: map[string]string{
				"machineconfiguration.openshift.io/role": role,
			},
		},
		Spec: mcfgv1.MachineConfigSpec{
			Config: ignitionRawExt,
		},
	}

	// Add the MachineConfig manifest to extra-manifests dir
	manifestBytes, err := yaml.Marshal(machineConfig)
	if err != nil {
		return err
	}
	manifestPath := fmt.Sprintf("%s/%s.yaml", extraManifestsPath, machineConfig.Name)
	manifestFile := ignasset.FileFromBytes(manifestPath, "root", 0644, manifestBytes)
	i.Config.Storage.Files = append(i.Config.Storage.Files, manifestFile)

	return nil
}
