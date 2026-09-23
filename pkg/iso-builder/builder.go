package isobuilder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-openapi/swag"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/appliance/pkg/asset/appliance"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/graph"
	"github.com/openshift/appliance/pkg/types"
	"github.com/openshift/installer/pkg/asset"
	assetstore "github.com/openshift/installer/pkg/asset/store"
)

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
	store, err := assetstore.NewStore(b.workingDir)
	if err != nil {
		return errors.Wrap(err, "failed to create asset store")
	}

	isoBuilderAssets := []asset.Asset{
		&config.ApplianceConfigProvider{
			Config: DefaultConfig(),
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

	outputISO := fmt.Sprintf(OutputISOPattern, outputArch)
	if err := b.renameOutput(outputISO); err != nil {
		return err
	}

	logrus.Infof("ISO created: %s", filepath.Join(b.workingDir, outputISO))
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

// DefaultConfig returns the hard-coded appliance configuration used for building.
func DefaultConfig() *types.ApplianceConfig {
	channel := graph.ReleaseChannelStable

	return &types.ApplianceConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: types.ApplianceConfigApiVersion,
			Kind:       "ApplianceConfig",
		},
		OcpRelease: types.ReleaseImage{
			Version:         "4.22",
			Channel:         &channel,
			CpuArchitecture: swag.String("x86_64"),
		},
		PullSecret:            readPullSecret(),
		DiskSizeGB:            swag.Int(200),
		StopLocalRegistry:     swag.Bool(false),
		EnableDefaultSources:  swag.Bool(false),
		UseDefaultSourceNames: swag.Bool(true),
		EnableInteractiveFlow: swag.Bool(true),
		SkipLocalRegistry:     swag.Bool(true),
		ImageRegistry: &types.ImageRegistry{
			UseBinary: swag.Bool(false),
		},
		AdditionalImages: &[]types.Image{
			{Name: "registry.redhat.io/rhel9/support-tools:latest"},
		},
		Operators: defaultOperators(),
	}
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

func defaultOperators() *[]types.Operator {
	pkg := func(name, channel string) types.IncludePackage {
		return types.IncludePackage{
			Name:     name,
			Channels: []types.IncludeChannel{{Name: channel}},
		}
	}

	return &[]types.Operator{
		{
			Catalog: "registry.redhat.io/redhat/redhat-operator-index:v4.22",
			IncludeConfig: types.IncludeConfig{
				Packages: []types.IncludePackage{
					pkg("kubevirt-hyperconverged", "stable"),
					pkg("mtv-operator", "release-v2.12"),
					pkg("kubernetes-nmstate-operator", "stable"),
					pkg("node-healthcheck-operator", "stable"),
					pkg("node-maintenance-operator", "stable"),
					pkg("fence-agents-remediation", "stable"),
					pkg("cluster-kube-descheduler-operator", "stable"),
					pkg("metallb-operator", "stable"),
					pkg("cluster-observability-operator", "stable"),
					pkg("redhat-oadp-operator", "stable"),
					pkg("local-storage-operator", "stable"),
					pkg("lvms-operator", "stable-4.22"),
					pkg("numaresources-operator", "4.22"),
					pkg("loki-operator", "stable-6.6"),
					pkg("cluster-logging", "stable-6.6"),
				},
			},
		},
	}
}
