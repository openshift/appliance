package ignition

import (
	"fmt"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/openshift/installer/pkg/asset/ignition"
)

// interactiveFlowIgnition takes care of generating the additional
// igntion files required to support the interactive flow.
type interactiveFlowIgnition struct {
	releaseVersion string
}

func NewInteractiveFlowIgnition(releaseVersion string) *interactiveFlowIgnition {
	return &interactiveFlowIgnition{
		releaseVersion: releaseVersion,
	}
}

func (i *interactiveFlowIgnition) AppendToIgnition(ign *igntypes.Config) {
	i.appendControlFiles(ign)
	i.appendInternalReleaseImageManifest(ign)
}

func (i *interactiveFlowIgnition) appendControlFiles(ign *igntypes.Config) {
	interactiveUIFile := ignition.FileFromString("/etc/assisted/interactive-ui", "root", 0644, "")
	ign.Storage.Files = append(ign.Storage.Files, interactiveUIFile)

	// Explicitly disable the load-config-iso service, not required in the OVE flow
	// (even though disabled by default, the udev rule may require it).
	noConfigImageFile := ignition.FileFromString("/etc/assisted/no-config-image", "root", 0644, "")
	ign.Storage.Files = append(ign.Storage.Files, noConfigImageFile)
}

func (i *interactiveFlowIgnition) appendInternalReleaseImageManifest(ign *igntypes.Config) {
	// Trim ocp bundle names longer than 64 chars.
	ocpBundleStr := fmt.Sprintf("ocp-release-bundle-%s", i.releaseVersion)
	if len(ocpBundleStr) > 64 {
		ocpBundleStr = ocpBundleStr[:64]
	}

	iriContent := fmt.Sprintf(`apiVersion: machineconfiguration.openshift.io/v1alpha1
kind: InternalReleaseImage
metadata:
  name: cluster
spec:
  releases:
  - name: %s
`, ocpBundleStr)

	// Keep the filepath in sync with openshift/installer#10176 until the installer min storage will be more robust.
	iriFile := ignition.FileFromString("/etc/assisted/extra-manifests/internalreleaseimage.yaml", "root", 0644, iriContent)
	ign.Storage.Files = append(ign.Storage.Files, iriFile)
}
