package releasebundle

import "fmt"

const maxOCPBundleTagLen = 64

// ImageRepository is the repository path (namespace/name) for the empty bundle
// image in the appliance local registry, without host or port.
const ImageRepository = "openshift/release-bundles"

// Tag returns the OCP release bundle image tag for releaseVersion, matching
// InternalReleaseImage naming and the 64-character limit.
func Tag(releaseVersion string) string {
	s := fmt.Sprintf("ocp-release-bundle-%s", releaseVersion)
	if len(s) > maxOCPBundleTagLen {
		return s[:maxOCPBundleTagLen]
	}
	return s
}
