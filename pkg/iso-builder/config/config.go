package isobuilder

import "encoding/json"

// Config defines the iso-builder configuration format.
// Field names follow the OpenShift installer naming convention.
type Config struct {
	OpenshiftVersion      string            `json:"openshiftVersion,omitempty"`
	ReleaseImageURL       string            `json:"releaseImageURL,omitempty"`
	PullSecret            string            `json:"pullSecret"`
	SSHKey                []string          `json:"sshKey,omitempty"`
	Architecture          string            `json:"architecture,omitempty"`
	Proxy                 *Proxy            `json:"proxy,omitempty"`
	AdditionalTrustBundle string            `json:"additionalTrustBundle,omitempty"`
	AdditionalNTPServers  []string          `json:"additionalNTPServers,omitempty"`
	FIPS                  bool              `json:"fips,omitempty"`
	OLMOperators          []OLMOperator     `json:"olmOperators,omitempty"`
	RendezvousIP          string            `json:"rendezvousIP,omitempty"`
	NetworkConfig         []json.RawMessage `json:"networkConfig,omitempty"`
	AdditionalImages      []string          `json:"additionalImages,omitempty"`
	ExtraManifests        []ExtraManifest   `json:"extraManifests,omitempty"`
}

type Proxy struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

type OLMOperator struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Channel string `json:"channel,omitempty"`
}

type ExtraManifest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
