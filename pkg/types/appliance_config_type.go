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

	OcpRelease            ReleaseImage   `json:"ocpRelease"`
	DiskSizeGB            *int           `json:"diskSizeGb"`
	PullSecret            string         `json:"pullSecret"`
	SshKey                *string        `json:"sshKey"`
	UserCorePass          *string        `json:"userCorePass"`
	ImageRegistry         *ImageRegistry `json:"imageRegistry"`
	EnableDefaultSources  *bool          `json:"enableDefaultSources"`
	StopLocalRegistry     *bool          `json:"stopLocalRegistry"`
	CreatePinnedImageSets *bool          `json:"createPinnedImageSets"`
	AdditionalImages      *[]Image       `json:"additionalImages,omitempty"`
	Operators             *[]Operator    `json:"operators,omitempty"`
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

// Structs copied from oc-mirror: https://github.com/openshift/oc-mirror/blob/main/v2/pkg/api/v1alpha2/types_config.go

// Image contains image pull information.
type Image struct {
	// Name of the image. This should be an exact image pin (registry/namespace/name@sha256:<hash>)
	// but is not required to be.
	Name string `json:"name"`
}

// Operator defines the configuration for operator catalog mirroring.
type Operator struct {
	// Mirror specific operator packages, channels, and versions, and their dependencies.
	// If HeadsOnly is true, these objects are mirrored on top of heads of all channels.
	// Otherwise, only these specific objects are mirrored.
	IncludeConfig `json:",inline"`

	// Catalog image to mirror. This image must be pullable and available for subsequent
	// pulls on later mirrors.
	// This image should be an exact image pin (registry/namespace/name@sha256:<hash>)
	// but is not required to be.
	Catalog string `json:"catalog"`
}

// IncludeConfig defines a list of packages for
// operator version selection.
type IncludeConfig struct {
	// Packages to include.
	Packages []IncludePackage `json:"packages" yaml:"packages"`
}

// IncludePackage contains a name (required) and channels and/or versions
// (optional) to include in the diff. The full package is only included if no channels
// or versions are specified.
type IncludePackage struct {
	// Name of package.
	Name string `json:"name" yaml:"name"`
	// Channels to include.
	Channels []IncludeChannel `json:"channels,omitempty" yaml:"channels,omitempty"`

	// All channels containing these bundles are parsed for an upgrade graph.
	IncludeBundle `json:",inline"`
}

// IncludeChannel contains a name (required) and versions (optional)
// to include in the diff. The full channel is only included if no versions are specified.
type IncludeChannel struct {
	// Name of channel.
	Name string `json:"name" yaml:"name"`

	IncludeBundle `json:",inline"`
}

// IncludeBundle contains a name (required) and versions (optional) to
// include in the diff. The full package or channel is only included if no
// versions are specified.
type IncludeBundle struct {
	// MinVersion to include, plus all versions in the upgrade graph to the MaxVersion.
	MinVersion string `json:"minVersion,omitempty" yaml:"minVersion,omitempty"`
	// MaxVersion to include as the channel head version.
	MaxVersion string `json:"maxVersion,omitempty" yaml:"maxVersion,omitempty"`
	// MinBundle to include, plus all bundles in the upgrade graph to the channel head.
	// Set this field only if the named bundle has no semantic version metadata.
	MinBundle string `json:"minBundle,omitempty" yaml:"minBundle,omitempty"`
}
