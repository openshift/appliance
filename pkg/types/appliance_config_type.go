package types

import (
	"github.com/containers/image/pkg/sysregistriesv2"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"
	"github.com/openshift/installer/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/appliance/pkg/consts"
	"github.com/openshift/appliance/pkg/graph"
)

// ApplianceConfigApiVersion is the version supported by this package.
const ApplianceConfigApiVersion = "v1beta1"

type ApplianceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	OcpRelease    ReleaseImage   `json:"ocpRelease"`
	DiskSizeGB    int            `json:"diskSizeGb"`
	PullSecret    string         `json:"pullSecret"`
	SshKey        *string        `json:"sshKey"`
	UserCorePass  *string        `json:"userCorePass"`
	ImageRegistry *ImageRegistry `json:"imageRegistry"`
	// ImageDigestSources lists sources/repositories for the release-image content.
	ImageDigestSources []types.ImageDigestSource `json:"imageDigestSources,omitempty"`
}

type ReleaseImage struct {
	Version         string                `json:"version"`
	Channel         *graph.ReleaseChannel `json:"channel"`
	CpuArchitecture *string               `json:"cpuArchitecture"`
	URL             *string               `json:"url"`
}

type ImageRegistry struct {
	URI  *string `json:"uri"`
	Port *int    `json:"port"`
}

func ConvertToTOMLRegistries(digestMirrorSources []types.ImageDigestSource) *sysregistriesv2.V2RegistriesConf {
	TOMLRegistries := &sysregistriesv2.V2RegistriesConf{
		Registries: []sysregistriesv2.Registry{},
	}

	for _, group := range bootstrap.MergedMirrorSets(digestMirrorSources) {
		if len(group.Mirrors) == 0 {
			continue
		}

		registry := sysregistriesv2.Registry{}
		registry.Endpoint.Location = group.Source
		registry.MirrorByDigestOnly = consts.RegistryMirrorByDigestOnly
		for _, mirror := range group.Mirrors {
			registry.Mirrors = append(registry.Mirrors, sysregistriesv2.Endpoint{Location: mirror})
		}
		TOMLRegistries.Registries = append(TOMLRegistries.Registries, registry)
	}

	return TOMLRegistries
}
