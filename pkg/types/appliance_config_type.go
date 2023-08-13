package types

import (
	"github.com/openshift/appliance/pkg/graph"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplianceConfigApiVersion is the version supported by this package.
const ApplianceConfigApiVersion = "v1beta1"

type ApplianceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	OcpRelease    ReleaseImage   `json:"ocpRelease"`
	DiskSizeGB    *int           `json:"diskSizeGb"`
	PullSecret    string         `json:"pullSecret"`
	SshKey        *string        `json:"sshKey"`
	UserCorePass  *string        `json:"userCorePass"`
	ImageRegistry *ImageRegistry `json:"imageRegistry"`
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
