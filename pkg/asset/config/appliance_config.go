package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-openapi/swag"
	"github.com/hashicorp/go-version"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/validate"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/graph"
	"github.com/openshift/appliance/pkg/types"
)

const (
	ApplianceConfigFilename       = "appliance-config.yaml"
	CustomClusterManifestsDir     = "openshift"
	CustomClusterManifestsPattern = "*.yaml"

	// CPU architectures
	CpuArchitectureX86     = "x86_64"
	CpuArchitectureAARCH64 = "aarch64"
	CpuArchitecturePPC64le = "ppc64le"

	// Release architecture
	ReleaseArchitectureAMD64   = "amd64"
	ReleaseArchitectureARM64   = "arm64"
	ReleaseArchitecturePPC64le = "ppc64le"

	// Validation values
	MinDiskSize     = 150
	RegistryMinPort = 1024
	RegistryMaxPort = 65535

	// Validation commands
	PodmanPull = "podman pull %s"
)

var (
	cpuArchitectures             = []string{CpuArchitectureX86, CpuArchitectureAARCH64, CpuArchitecturePPC64le}
	releaseImage, releaseVersion string
)

// ApplianceConfig reads the appliance-config.yaml file.
type ApplianceConfig struct {
	File     *asset.File
	Config   *types.ApplianceConfig
	Template string
}

var _ asset.WritableAsset = (*ApplianceConfig)(nil)

// Name returns a human friendly name for the asset.
func (*ApplianceConfig) Name() string {
	return "Appliance Config"
}

// Dependencies returns all the dependencies directly needed to generate
// the asset.
func (*ApplianceConfig) Dependencies() []asset.Asset {
	return []asset.Asset{}
}

// Generate generates the Agent Config manifest.
func (a *ApplianceConfig) Generate(_ context.Context, dependencies asset.Parents) error {
	applianceConfigTemplate := `#
# Note: This is a sample ApplianceConfig file showing
# which fields are available to aid you in creating your
# own appliance-config.yaml file.
#
apiVersion: %s
kind: ApplianceConfig
ocpRelease:
  # OCP release version in major.minor or major.minor.patch format
  # (in case of major.minor - latest patch version will be used)
  # If the specified version is not yet available, the latest supported version will be used.
  version: ocp-release-version
  # OCP release update channel: stable|fast|eus|candidate
  # Default: %s
  # [Optional]
  channel: ocp-release-channel
  # OCP release CPU architecture: x86_64|aarch64|ppc64le
  # Default: %s
  # [Optional]
  cpuArchitecture: cpu-architecture
# If specified, should be at least %dGiB.
# If not specified, the disk image should be resized when 
# cloning to a device (e.g. using virt-resize tool).
# [Optional]
diskSizeGB: disk-size
# PullSecret required for mirroring the OCP release payload
pullSecret: pull-secret
# Public SSH key for accessing the appliance during the bootstrap phase
# [Optional]
sshKey: ssh-key
# Password of user 'core' for connecting from console
# [Optional]
userCorePass: user-core-pass
# Local image registry details (used when building the appliance)
# [Optional]
imageRegistry:
  # The URI for the image
  # Default: %s
  # Alternative: quay.io/libpod/registry:2.8
  # [Optional]
  uri: uri
  # The image registry container TCP port to bind. A valid port number is between %d and %d.
  # Default: %d
  # [Optional]
  port: port
# Enable all default CatalogSources (on openshift-marketplace namespace).
# Should be disabled for disconnected environments.
# Default: false
# [Optional]
enableDefaultSources: %t
# Stop the local registry post cluster installation.
# Note that additional images and operators won't be available when stopped.
# Default: false
# [Optional]
stopLocalRegistry: %t
# Additional images to be included in the appliance disk image.
# [Optional]
# additionalImages:
#   - name: image-url
# Operators to be included in the appliance disk image.
# See examples in https://github.com/openshift/oc-mirror/blob/main/docs/imageset-config-ref.yaml.
# [Optional]
# operators:
# - catalog: catalog-uri
#   packages:
#     - name: package-name
#       channels:
#         - name: channel-name
`
	a.Template = fmt.Sprintf(
		applianceConfigTemplate,
		types.ApplianceConfigApiVersion, graph.ReleaseChannelStable, CpuArchitectureX86,
		MinDiskSize, consts.RegistryImage, RegistryMinPort, RegistryMaxPort, consts.RegistryPort,
		consts.EnableDefaultSources, consts.StopLocalRegistry)

	return nil
}

// PersistToFile writes the appliance-config.yaml file to the assets folder
func (a *ApplianceConfig) PersistToFile(directory string) error {
	if a.Template == "" {
		return nil
	}

	templatePath := filepath.Join(directory, ApplianceConfigFilename)
	templateByte := []byte(a.Template)
	err := os.WriteFile(templatePath, templateByte, 0644) // #nosec G306
	if err != nil {
		return err
	}

	return nil
}

// Files returns the files generated by the asset.
func (a *ApplianceConfig) Files() []*asset.File {
	if a.File != nil {
		return []*asset.File{a.File}
	}
	return []*asset.File{}
}

// Load returns agent config asset from the disk.
func (a *ApplianceConfig) Load(f asset.FileFetcher) (bool, error) {
	file, err := f.FetchByName(ApplianceConfigFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrap(err, fmt.Sprintf("failed to load %s file", ApplianceConfigFilename))
	}

	config := &types.ApplianceConfig{}
	if err = yaml.UnmarshalStrict(file.Data, config); err != nil {
		// Log full error only on debug level
		logrus.Debug(err)

		// Search for failed to parse field
		r := regexp.MustCompile(`field .\S*`)
		field := r.FindString(err.Error())
		if field != "" {
			field = fmt.Sprintf(" (error in %s)", field)
		}

		return false, errors.New(fmt.Sprintf("can't parse %s. Ensure the config file is configured correctly%s. For additional info add '--log-level debug'.", ApplianceConfigFilename, field))
	}

	a.File, a.Config = file, config

	if err = a.validateConfig(f).ToAggregate(); err != nil {
		return false, errors.Wrapf(err, "invalid Appliance Config configuration")
	}

	// Fallback to x86_64
	if config.OcpRelease.CpuArchitecture == nil {
		config.OcpRelease.CpuArchitecture = swag.String(CpuArchitectureX86)
	}

	cpuArch := strings.ToLower(*config.OcpRelease.CpuArchitecture)
	if !funk.Contains(cpuArchitectures, cpuArch) {
		return false, errors.Errorf("Unsupported CPU architecture: %s", cpuArch)
	}
	config.OcpRelease.CpuArchitecture = swag.String(cpuArch)

	releaseImage, releaseVersion, err = a.getRelease()
	if err != nil {
		return false, err
	}
	config.OcpRelease.URL = &releaseImage
	config.OcpRelease.Version = releaseVersion

	if config.ImageRegistry == nil {
		config.ImageRegistry = &types.ImageRegistry{
			URI:  swag.String(consts.RegistryImage),
			Port: swag.Int(consts.RegistryPort),
		}
	} else {
		if config.ImageRegistry.URI == nil {
			config.ImageRegistry.URI = swag.String(consts.RegistryImage)
		}
		if config.ImageRegistry.Port == nil {
			config.ImageRegistry.Port = swag.Int(consts.RegistryPort)
		}
	}

	return true, nil
}

func (a *ApplianceConfig) GetCpuArchitecture() string {
	// Note: in Load func, we ensure that CpuArchitecture is not nil and fallback to x86_64
	return swag.StringValue(a.Config.OcpRelease.CpuArchitecture)
}

func GetReleaseArchitectureByCPU(arch string) string {
	switch arch {
	case CpuArchitectureX86:
		return ReleaseArchitectureAMD64
	case CpuArchitectureAARCH64:
		return ReleaseArchitectureARM64
	default:
		return arch
	}
}

func (a *ApplianceConfig) getRelease() (string, string, error) {
	if releaseImage != "" && releaseVersion != "" {
		// Return cached values
		return releaseImage, releaseVersion, nil
	}

	graphConfig := graph.GraphConfig{
		Arch:    GetReleaseArchitectureByCPU(*a.Config.OcpRelease.CpuArchitecture),
		Version: a.Config.OcpRelease.Version,
		Channel: a.Config.OcpRelease.Channel,
	}

	g := graph.NewGraph(graphConfig)
	releaseImage, releaseVersion, err := g.GetReleaseImage()
	if err != nil {
		return "", "", err
	}
	return releaseImage, releaseVersion, nil
}

func (a *ApplianceConfig) validateConfig(f asset.FileFetcher) field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate apiVersion
	if err := a.validateApiVersion(); err != nil {
		allErrs = append(allErrs, err...)
	}

	// Validate ocpRelease
	if err := a.validateOcpRelease(); err != nil {
		allErrs = append(allErrs, err...)
	}

	// Validate diskSizeGB
	if err := a.validateDiskSize(); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("diskSizeGB"), a.Config.DiskSizeGB, err.Error()))
	}

	// Validate imageRegistry
	if err := a.validateImageRegistry(); err != nil {
		allErrs = append(allErrs, err...)
	}

	// Validate pullSecret
	if err := validate.ImagePullSecret(a.Config.PullSecret); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("pullSecret"), a.Config.PullSecret, err.Error()))
	}

	// Validate sshKey
	if a.Config.SshKey != nil {
		if err := validate.SSHPublicKey(*a.Config.SshKey); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("sshKey"), *a.Config.SshKey, err.Error()))
		}
	}

	return allErrs
}

func (a *ApplianceConfig) validateImageRegistry() field.ErrorList {
	allErrs := field.ErrorList{}

	if a.Config.ImageRegistry == nil {
		return nil
	}

	if a.Config.ImageRegistry.URI != nil {
		cmd := fmt.Sprintf(PodmanPull, swag.StringValue(a.Config.ImageRegistry.URI))
		logrus.Debugf("Running uri validation cmd: %s", cmd)
		if _, err := executer.NewExecuter().Execute(cmd); err != nil {
			allErrs = append(allErrs, field.ErrorList{field.Invalid(field.NewPath("imageRegistry.uri"),
				swag.StringValue(a.Config.ImageRegistry.URI),
				fmt.Sprintf("Invalid uri: %s", err.Error()))}...)
		}
	}

	if a.Config.ImageRegistry.Port != nil {
		registryPort := swag.IntValue(a.Config.ImageRegistry.Port)
		if registryPort < RegistryMinPort || registryPort > RegistryMaxPort {
			allErrs = append(allErrs, field.ErrorList{field.Invalid(field.NewPath("imageRegistry.port"),
				swag.IntValue(a.Config.ImageRegistry.Port),
				fmt.Sprintf("registryPort must be between %d and %d", RegistryMinPort, RegistryMaxPort))}...)
		}
	}
	return allErrs
}

func (a *ApplianceConfig) validateApiVersion() field.ErrorList {
	if a.Config.TypeMeta.APIVersion == "" {
		return field.ErrorList{field.Required(field.NewPath("apiVersion"), "apiVersion is required")}
	}
	switch v := a.Config.APIVersion; v {
	case types.ApplianceConfigApiVersion: // Current version
	default:
		return field.ErrorList{field.Invalid(field.NewPath("apiVersion"),
			a.Config.TypeMeta.APIVersion,
			fmt.Sprintf("apiVersion must be %q", types.ApplianceConfigApiVersion))}
	}
	return nil
}

func (a *ApplianceConfig) validateOcpRelease() field.ErrorList {
	allErrs := field.ErrorList{}

	// Validate ocpRelease.version
	if a.Config.OcpRelease.Version == "" {
		allErrs = append(allErrs, field.ErrorList{field.Required(field.NewPath("ocpRelease.version"),
			"ocpRelease version is required")}...)
	}
	minOcpVer, _ := version.NewVersion(consts.MinOcpVersion)
	ocpVer, err := version.NewVersion(a.Config.OcpRelease.Version)
	if err != nil {
		allErrs = append(allErrs, field.ErrorList{field.Invalid(field.NewPath("ocpRelease.version"),
			a.Config.OcpRelease.Version,
			fmt.Sprintf("OCP release version must be in major.minor or major.minor.patch format %q", err))}...)
	} else if ocpVer.LessThan(minOcpVer) {
		allErrs = append(allErrs, field.ErrorList{field.Invalid(field.NewPath("ocpRelease.version"),
			a.Config.OcpRelease.Version,
			fmt.Sprintf("OCP release version must be at least %s", consts.MinOcpVersion))}...)
	}

	// Validate ocpRelease.channel
	if a.Config.OcpRelease.Channel != nil {
		switch *a.Config.OcpRelease.Channel {
		case graph.ReleaseChannelStable:
		case graph.ReleaseChannelFast:
		case graph.ReleaseChannelCandidate:
		case graph.ReleaseChannelEUS:
		default:
			allErrs = append(allErrs, field.ErrorList{field.Invalid(field.NewPath("ocpRelease.channel"),
				a.Config.OcpRelease.Channel,
				"Unsupported OCP release channel (supported channels: stable|fast|eus|candidate)")}...)
		}
	} else {
		channel := graph.ReleaseChannelStable
		a.Config.OcpRelease.Channel = &channel
	}

	// Validate ocpRelease.cpuArchitecture
	if swag.StringValue(a.Config.OcpRelease.CpuArchitecture) != "" {
		switch *a.Config.OcpRelease.CpuArchitecture {
		case CpuArchitectureX86:
		case CpuArchitectureAARCH64:
		case CpuArchitecturePPC64le:
		default:
			allErrs = append(allErrs, field.ErrorList{field.Invalid(field.NewPath("ocpRelease.cpuArchitecture"),
				a.Config.OcpRelease.CpuArchitecture,
				"Unsupported OCP release cpu architecture (supported architectures: x86_64|aarch64|ppc64le)")}...)
		}
	}

	return allErrs
}

func (a *ApplianceConfig) validateDiskSize() error {
	if a.Config.DiskSizeGB == nil {
		return nil
	}
	if *a.Config.DiskSizeGB < MinDiskSize {
		return fmt.Errorf("diskSizeGB must be at least %d GiB", MinDiskSize)
	}
	return nil
}
