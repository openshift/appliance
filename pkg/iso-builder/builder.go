package isobuilder

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-openapi/swag"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/appliance/pkg/asset/appliance"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/graph"
	isobuilderconfig "github.com/openshift/appliance/pkg/iso-builder/config"
	"github.com/openshift/appliance/pkg/types"
	"github.com/openshift/installer/pkg/asset"
	assetstore "github.com/openshift/installer/pkg/asset/store"
)

//go:generate go run ./gen_embed_area

//go:embed config_embed_area.bin
var rawConfigArea string

const (
	// OutputISOPattern is the naming pattern for the generated ISO file.
	OutputISOPattern = "agent.%s.iso"
	outputArch       = "x86_64"
)

// Builder orchestrates the ISO build process.
type Builder struct {
	workingDir string
}

// NewBuilder creates a Builder that writes artifacts to workingDir.
func NewBuilder(workingDir string) *Builder {
	return &Builder{workingDir: workingDir}
}

// Build generates the installation ISO using the embedded configuration.
func (b *Builder) Build(ctx context.Context) error {
	embeddedCfg, err := b.loadEmbeddedConfig()
	if err != nil {
		logrus.Warn("No embedded configuration found, using built-in defaults")
		embeddedCfg = defaultISOBuilderConfig()
	}
	logrus.Infof("Configuration loaded: version=%s arch=%s", embeddedCfg.OpenshiftVersion, embeddedCfg.Architecture)

	if err := b.applyLiveISOBuilderAsset(ctx, embeddedCfg); err != nil {
		return err
	}

	outputISO := fmt.Sprintf(OutputISOPattern, outputArch)
	if err := b.renameOutput(outputISO); err != nil {
		return err
	}

	logrus.Infof("ISO created: %s", filepath.Join(b.workingDir, outputISO))
	return nil
}

// Not yet converted: Proxy, AdditionalTrustBundle, AdditionalNTPServers,
// RendezvousIP, NetworkConfig, ExtraManifests. These fields target the
// install-config / agent-config pipeline and will be addressed separately.
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
		DiskSizeGB:            swag.Int(200),
		StopLocalRegistry:     swag.Bool(false),
		EnableDefaultSources:  swag.Bool(false),
		UseDefaultSourceNames: swag.Bool(true),
		EnableInteractiveFlow: swag.Bool(true),
		SkipLocalRegistry:     swag.Bool(true),
		ImageRegistry: &types.ImageRegistry{
			UseBinary: swag.Bool(false),
		},
	}

	if cfg.Architecture != "" {
		appCfg.OcpRelease.CpuArchitecture = swag.String(cfg.Architecture)
	}

	if cfg.ReleaseImageURL != "" {
		appCfg.OcpRelease.URL = swag.String(cfg.ReleaseImageURL)
	}

	if len(cfg.SSHKey) > 0 {
		appCfg.SshKey = swag.String(strings.Join(cfg.SSHKey, "\n"))
	}

	if cfg.FIPS {
		appCfg.EnableFips = swag.Bool(true)
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

func (b *Builder) loadEmbeddedConfig() (*isobuilderconfig.Config, error) {
	return isobuilderconfig.ReadFromData([]byte(rawConfigArea))
}

func readPullSecret() string {
	path := os.Getenv("PULL_SECRET_FILE")
	if path == "" {
		logrus.Warn("PULL_SECRET_FILE is not set, using empty pull secret placeholder")
		return `{"auths":{}}`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		logrus.Fatalf("Failed to read pull secret from PULL_SECRET_FILE (%s): %v", path, err)
	}
	return string(data)
}
