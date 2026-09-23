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
	// Cluster proxy settings (http/https/noProxy).
	Proxy *Proxy `json:"proxy,omitempty"`
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
	NetworkConfig []json.RawMessage `json:"networkConfig,omitempty"`
	// Additional image pullspecs to be mirrored.
	AdditionalImages []string `json:"additionalImages,omitempty"`
	// Custom manifests to be added in the cluster.
	ExtraManifests []ExtraManifest `json:"extraManifests,omitempty"`
}

// Proxy holds cluster-wide proxy settings.
type Proxy struct {
	// HTTP proxy URL.
	HTTPProxy string `json:"httpProxy,omitempty"`
	// HTTPS proxy URL.
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	// Comma-separated list of destinations that should bypass the proxy.
	NoProxy string `json:"noProxy,omitempty"`
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

// ExtraManifest is a custom Kubernetes manifest to include in the cluster.
type ExtraManifest struct {
	// Manifest file name.
	Name string `json:"name"`
	// Raw manifest content.
	Content string `json:"content"`
}
