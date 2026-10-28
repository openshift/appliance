package isobuilder

import "encoding/json"

// Config defines the iso-builder configuration format.
// Field names follow the OpenShift installer naming convention.
type Config struct {
	// OCP version to be mirrored.
	OpenshiftVersion string `json:"openshiftVersion,omitempty"`
	// Pullspec for the release image; used instead of OpenshiftVersion.
	ReleaseImageURL string `json:"releaseImageURL,omitempty"`
	// Secret to use when pulling images.
	PullSecret string `json:"pullSecret"`
	// Public SSH keys to provide access to instances.
	SSHKey []string `json:"sshKey,omitempty"`
	// Cluster CPU architecture.
	Architecture string `json:"architecture,omitempty"`
	// PEM-encoded X.509 certificate bundle for the nodes trusted certificate store.
	AdditionalTrustBundle string `json:"additionalTrustBundle,omitempty"`
	// Additional NTP servers to use for provisioning.
	AdditionalNTPServers []string `json:"additionalNTPServers,omitempty"`
	// Enables FIPS mode.
	FIPS bool `json:"fips,omitempty"`
	// OLM operators to be mirrored.
	OLMOperators []OLMOperator `json:"olmOperators,omitempty"`
	// IP address used as rendezvous point.
	RendezvousIP string `json:"rendezvousIP,omitempty"`
	// NMState configs for static networking.
	NetworkConfig []NetworkConfigEntry `json:"networkConfig,omitempty"`
	// Additional image pullspecs to be mirrored.
	AdditionalImages []string `json:"additionalImages,omitempty"`
}

// OLMOperator identifies an OLM operator to be mirrored into the ISO.
type OLMOperator struct {
	// Operator package name.
	Name string `json:"name"`
	// Operator version.
	Version string `json:"version,omitempty"`
	// OLM channel to track.
	Channel string `json:"channel,omitempty"`
}

// NetworkConfigEntry defines static network configuration for a host.
type NetworkConfigEntry struct {
	// Interfaces is a list of interfaces with name and MAC address.
	Interfaces []Interface `json:"interfaces"`
	// Config contains the nmstate YAML configuration.
	Config json.RawMessage `json:"config"`
}

// Interface defines a network interface.
type Interface struct {
	// Name is the interface name.
	Name string `json:"name"`
	// MacAddress is the MAC address for the interface.
	MacAddress string `json:"macAddress"`
}
