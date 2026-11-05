package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-openapi/swag/conv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/appliance/pkg/asset/appliance"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/graph"
	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
	"github.com/openshift/appliance/pkg/iso-builder/embeddedconfig"
	"github.com/openshift/appliance/pkg/oc"
	"github.com/openshift/appliance/pkg/ocmirror"
	"github.com/openshift/appliance/pkg/tools"
	"github.com/openshift/appliance/pkg/types"
	"github.com/openshift/installer/pkg/asset"
	assetstore "github.com/openshift/installer/pkg/asset/store"
)

const (
	// OutputISOPattern is the naming pattern for the generated ISO file.
	OutputISOPattern = "agent.%s.iso"
	// OutputArch is the only architecture supported for ISO output.
	OutputArch = "x86_64"
)

// Builder orchestrates the ISO build process.
type Builder struct {
	workingDir       string
	additionalImages []string
}

// NewBuilder creates a Builder that writes artifacts to workingDir.
// Any additionalImages are appended to the embedded config at build time.
func NewBuilder(workingDir string, additionalImages []string) *Builder {
	return &Builder{workingDir: workingDir, additionalImages: additionalImages}
}

// Build generates the installation ISO using the embedded configuration.
func (b *Builder) Build(ctx context.Context) error {
	embeddedCfg, err := embeddedconfig.LoadConfig()
	if err != nil {
		return fmt.Errorf("no configuration has been found in this binary")
	}

	arch := embeddedCfg.Architecture
	if arch == "" {
		arch = OutputArch
	}
	logrus.Infof("Configuration loaded: version=%s arch=%s", embeddedCfg.OpenshiftVersion, arch)

	if len(b.additionalImages) > 0 {
		logrus.Infof("Adding %d additional image(s) from CLI flags", len(b.additionalImages))
		embeddedCfg.AdditionalImages = append(embeddedCfg.AdditionalImages, b.additionalImages...)
	}

	if err := initializeRegistryAuth(b.workingDir, embeddedCfg.PullSecret); err != nil {
		return err
	}

	if err := b.acquireTools(embeddedCfg); err != nil {
		return err
	}

	if err := b.applyLiveISOBuilderAsset(ctx, embeddedCfg); err != nil {
		return err
	}

	outputISO := fmt.Sprintf(OutputISOPattern, OutputArch)
	if err := b.renameOutput(outputISO); err != nil {
		return err
	}

	logrus.Infof("ISO created: %s", filepath.Join(b.workingDir, outputISO))
	return nil
}

func (b *Builder) convertToApplianceConfig(cfg *isobuilderconfig.Config) *types.ApplianceConfig {
	channel := graph.ReleaseChannelStable

	appCfg := &types.ApplianceConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: types.ApplianceConfigApiVersion,
			Kind:       "ApplianceConfig",
		},
		OcpRelease: types.ReleaseImage{
			Version: cfg.OpenshiftVersion,
			Channel: &channel,
		},
		PullSecret:            cfg.PullSecret,
		DiskSizeGB:            conv.Pointer(200),
		StopLocalRegistry:     conv.Pointer(false),
		EnableDefaultSources:  conv.Pointer(false),
		UseDefaultSourceNames: conv.Pointer(true),
		EnableInteractiveFlow: conv.Pointer(true),
		SkipLocalRegistry:     conv.Pointer(true),
		ImageRegistry: &types.ImageRegistry{
			UseBinary: conv.Pointer(true),
		},
	}

	if cfg.Architecture != "" {
		appCfg.OcpRelease.CpuArchitecture = conv.Pointer(cfg.Architecture)
	}

	if cfg.ReleaseImageURL != "" {
		appCfg.OcpRelease.URL = conv.Pointer(cfg.ReleaseImageURL)
	}

	if len(cfg.SSHKey) > 0 {
		appCfg.SshKey = conv.Pointer(strings.Join(cfg.SSHKey, "\n"))
	}

	if cfg.FIPS {
		appCfg.EnableFips = conv.Pointer(true)
	}

	if len(cfg.AdditionalImages) > 0 {
		images := make([]types.Image, len(cfg.AdditionalImages))
		for i, img := range cfg.AdditionalImages {
			images[i] = types.Image{Name: img}
		}
		appCfg.AdditionalImages = &images
	}

	if len(cfg.OLMOperators) > 0 {
		appCfg.Operators = convertOperators(cfg.OpenshiftVersion, cfg.OLMOperators)
	}

	if cfg.RendezvousIP != "" {
		appCfg.RendezvousIP = conv.Pointer(cfg.RendezvousIP)
	}

	if len(cfg.AdditionalNTPServers) > 0 {
		appCfg.AdditionalNTPSources = cfg.AdditionalNTPServers
	}

	appCfg.AgentPullSecret = cfg.PullSecret

	if cfg.AdditionalTrustBundle != "" {
		appCfg.AdditionalTrustBundle = cfg.AdditionalTrustBundle
	}

	if len(cfg.NetworkConfig) > 0 {
		networkConfig := make([]types.NetworkConfigEntry, len(cfg.NetworkConfig))
		for i, entry := range cfg.NetworkConfig {
			interfaces := make([]types.NetworkInterface, len(entry.Interfaces))
			for j, iface := range entry.Interfaces {
				interfaces[j] = types.NetworkInterface{
					Name:       iface.Name,
					MacAddress: iface.MacAddress,
				}
			}
			networkConfig[i] = types.NetworkConfigEntry{
				Interfaces: interfaces,
				Config:     entry.Config,
			}
		}
		appCfg.NetworkConfig = networkConfig
	}

	return appCfg
}

func convertOperators(openshiftVersion string, olmOps []isobuilderconfig.OLMOperator) *[]types.Operator {
	catalog := fmt.Sprintf("registry.redhat.io/redhat/redhat-operator-index:v%s", openshiftVersion)

	packages := make([]types.IncludePackage, len(olmOps))
	for i, op := range olmOps {
		pkg := types.IncludePackage{Name: op.Name}
		if op.Channel != "" {
			pkg.Channels = []types.IncludeChannel{{Name: op.Channel}}
		}
		if op.Version != "" {
			pkg.IncludeBundle = types.IncludeBundle{MinVersion: op.Version}
		}
		packages[i] = pkg
	}

	return &[]types.Operator{{
		Catalog:       catalog,
		IncludeConfig: types.IncludeConfig{Packages: packages},
	}}
}

// acquireTools downloads the external tools into a "tools" folder under the working
// directory and prepends it to PATH, so the legacy builder can resolve them from
// the environment.
func (b *Builder) acquireTools(isoBuilderConfig *isobuilderconfig.Config) error {
	releaseVersion, err := resolveReleaseVersion(isoBuilderConfig)
	if err != nil {
		return errors.Wrap(err, "failed to resolve OCP release version")
	}

	supportedArch, err := validateArch()
	if err != nil {
		return err
	}

	toolsDir := filepath.Join(b.workingDir, "tools")
	if err := tools.CreateToolsDir(toolsDir); err != nil {
		return err
	}

	toolConfig := tools.ToolConfig{
		ReleaseVersion: releaseVersion,
		ToolsDir:       toolsDir,
		SupportedArch:  supportedArch,
	}

	if _, err := oc.NewClient(oc.ClientConfig{ToolConfig: toolConfig}).Acquire(); err != nil {
		return errors.Wrap(err, "failed to acquire oc client")
	}

	if _, err := ocmirror.NewMirror(ocmirror.MirrorConfig{ToolConfig: toolConfig}).Acquire(); err != nil {
		return errors.Wrap(err, "failed to acquire oc-mirror")
	}

	return prependToPath(toolsDir)
}

// resolveReleaseVersion returns the full OCP version used to pick the matching oc
// client. When release image URL is set, the configured version is used as-is.
func resolveReleaseVersion(cfg *isobuilderconfig.Config) (string, error) {
	if cfg.ReleaseImageURL != "" {
		return cfg.OpenshiftVersion, nil
	}

	channel := graph.ReleaseChannelStable
	_, version, err := graph.NewGraph(graph.GraphConfig{
		Arch:    "amd64",
		Version: cfg.OpenshiftVersion,
		Channel: &channel,
	}).GetReleaseImage()
	if err != nil {
		return "", err
	}
	return version, nil
}

// prependToPath puts dir at the front of the PATH environment variable.
func prependToPath(dir string) error {
	path := os.Getenv("PATH")
	if path != "" {
		dir = dir + string(os.PathListSeparator) + path
	}
	return os.Setenv("PATH", dir)
}

// This method is used to clearly mark the adoption of the legacy code.
func (b *Builder) applyLiveISOBuilderAsset(ctx context.Context, isoBuilderConfig *isobuilderconfig.Config) error {
	store, err := assetstore.NewStore(b.workingDir)
	if err != nil {
		return errors.Wrap(err, "failed to create asset store")
	}

	isoBuilderAssets := []asset.Asset{
		&config.ApplianceConfigProvider{
			Config: b.convertToApplianceConfig(isoBuilderConfig),
		},
		&config.EnvConfig{
			AssetsDir: b.workingDir,
			IsLiveISO: true,
		},
		&appliance.ApplianceLiveISO{},
	}
	for _, a := range isoBuilderAssets {
		if err := store.Fetch(ctx, a); err != nil {
			return errors.Wrapf(err, "failed to fetch %s", a.Name())
		}
	}

	return nil
}

func (b *Builder) renameOutput(outputISO string) error {
	src := filepath.Join(b.workingDir, consts.ApplianceLiveIsoFileName)
	dst := filepath.Join(b.workingDir, outputISO)
	if err := os.Rename(src, dst); err != nil {
		return errors.Wrapf(err, "failed to rename %s to %s", src, dst)
	}
	return nil
}

// validateArch returns the mirror.openshift.com arch path segment for
// tool acquisition (oc and oc-mirror). Both mac platforms (amd64 and arm64) are
// published under the x86_64 path on the mirror.
func validateArch() (string, error) {
	supported := (runtime.GOOS == "linux" && runtime.GOARCH == "amd64") ||
		(runtime.GOOS == "darwin" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"))
	if supported {
		return OutputArch, nil
	}
	return "", fmt.Errorf("unsupported platform/arch: %s/%s", runtime.GOOS, runtime.GOARCH)
}

// initializeRegistryAuth sets up credentials for oc and oc-mirror to find automatically.
// It creates XDG_RUNTIME_DIR in the working directory and writes the pull secret to
// ${XDG_RUNTIME_DIR}/containers/auth.json, isolating auth credentials to the build's working directory.
func initializeRegistryAuth(workingDir, pullSecret string) error {
	if pullSecret == "" {
		return nil
	}

	// Create a temporary XDG_RUNTIME_DIR in the working directory to isolate auth
	xdgRuntime, err := filepath.Abs(filepath.Join(workingDir, "xdg-runtime-dir"))
	if err != nil {
		return errors.Wrap(err, "failed to get absolute path for XDG_RUNTIME_DIR")
	}

	if err := os.MkdirAll(xdgRuntime, 0o700); err != nil {
		return errors.Wrap(err, "failed to create XDG_RUNTIME_DIR")
	}

	if err := os.Setenv("XDG_RUNTIME_DIR", xdgRuntime); err != nil {
		return errors.Wrap(err, "failed to set XDG_RUNTIME_DIR")
	}
	logrus.Debugf("Set XDG_RUNTIME_DIR=%s", xdgRuntime)

	authDir := filepath.Join(xdgRuntime, "containers")
	authFile := filepath.Join(authDir, "auth.json")

	if err := os.MkdirAll(authDir, 0o700); err != nil {
		return errors.Wrap(err, "failed to create auth directory")
	}
	if err := os.WriteFile(authFile, []byte(pullSecret), 0o600); err != nil {
		return errors.Wrap(err, "failed to write auth.json")
	}
	logrus.Debugf("Wrote pull secret to %s", authFile)
	return nil
}
