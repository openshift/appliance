package manifests

import (
	"fmt"

	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/installer/pkg/asset"
)

var (
	clusterImageSetFilename = "cluster-image-set.yaml"
)

// ClusterImageSet generates the cluster-image-set.yaml file.
type ClusterImageSet struct {
	File   *asset.File
	Config *hivev1.ClusterImageSet
}

var _ asset.Asset = (*ClusterImageSet)(nil)

// Name returns a human friendly name for the asset.
func (*ClusterImageSet) Name() string {
	return "ClusterImageSet Config"
}

// Dependencies returns all the dependencies directly needed to generate
// the asset.
func (*ClusterImageSet) Dependencies() []asset.Asset {
	return []asset.Asset{
		&config.ApplianceConfig{},
	}
}

// Generate generates the ClusterImageSet manifest.
func (a *ClusterImageSet) Generate(dependencies asset.Parents) error {
	applianceConfig := &config.ApplianceConfig{}
	dependencies.Get(applianceConfig)

	clusterImageSet := &hivev1.ClusterImageSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("openshift-%s", applianceConfig.Config.OcpRelease.Version),
		},
		Spec: hivev1.ClusterImageSetSpec{
			ReleaseImage: *applianceConfig.Config.OcpRelease.URL,
		},
	}
	a.Config = clusterImageSet

	configData, err := yaml.Marshal(clusterImageSet)
	if err != nil {
		return errors.Wrap(err, "failed to marshal agent cluster image set")
	}

	a.File = &asset.File{
		Filename: clusterImageSetFilename,
		Data:     configData,
	}

	return nil
}
