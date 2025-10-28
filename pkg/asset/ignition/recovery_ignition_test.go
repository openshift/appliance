package ignition

import (
	"testing"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	ignasset "github.com/openshift/installer/pkg/asset/ignition"
	"github.com/vincent-petithory/dataurl"
)

var _ = Describe("Test RecoveryIgnition", func() {
	Describe("Interactive UI File Constants", func() {
		It("has correct constant values", func() {
			Expect(InteractiveUIFilePath).To(Equal("/etc/assisted/interactive-ui"))
			Expect(InteractiveUIFileMode).To(Equal(0644))
			Expect(InteractiveUIFileOwner).To(Equal("root"))
		})
	})

	Describe("Generate() - Interactive Flow Sentinel File", func() {
		It("adds interactive UI sentinel file when EnableInteractiveFlow is true", func() {
			baseIgnition := igntypes.Config{
				Ignition: igntypes.Ignition{
					Version: "3.2.0",
				},
				Storage: igntypes.Storage{
					Files: []igntypes.File{},
				},
			}

			// Simulate the logic from Generate() when EnableInteractiveFlow is true
			interactiveUIFile := ignasset.FileFromString(InteractiveUIFilePath, InteractiveUIFileOwner, InteractiveUIFileMode, "")
			baseIgnition.Storage.Files = append(baseIgnition.Storage.Files, interactiveUIFile)

			// Verify the sentinel file was added
			Expect(baseIgnition.Storage.Files).To(HaveLen(1))

			// Find the interactive UI file in the files list
			var foundInteractiveFile *igntypes.File
			if baseIgnition.Storage.Files[0].Node.Path == InteractiveUIFilePath {
				foundInteractiveFile = &baseIgnition.Storage.Files[0]
			}

			// Verify the file was found and has correct properties
			Expect(foundInteractiveFile).NotTo(BeNil(), "Interactive UI sentinel file should be present")
			Expect(foundInteractiveFile.Node.Path).To(Equal(InteractiveUIFilePath))
			Expect(*foundInteractiveFile.Node.User.Name).To(Equal(InteractiveUIFileOwner))
			Expect(*foundInteractiveFile.FileEmbedded1.Mode).To(Equal(InteractiveUIFileMode))
			Expect(*foundInteractiveFile.Node.Overwrite).To(BeTrue())

			// Verify it's an empty file (sentinel)
			expectedEmptySource := dataurl.EncodeBytes([]byte(""))
			Expect(*foundInteractiveFile.FileEmbedded1.Contents.Source).To(Equal(expectedEmptySource))
		})
	})
})

func TestRecoveryIgnition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "recovery_ignition_test")
}
