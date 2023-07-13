package upgrade

// Metadata describes an upgrade bundle. This will be serialized to JSON and added to the tar
// archive as the first item, named `metadata.json`.
type Metadata struct {
	Version string   `json:"version,omitempty"`
	Arch    string   `json:"arch,omitempty"`
	Release string   `json:"release,omitempty"`
	Images  []string `json:"images,omitempty"`
}
