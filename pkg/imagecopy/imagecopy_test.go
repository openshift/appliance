package imagecopy

import (
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test imagecopy", func() {
	It("CopyToFile fails with invalid image URL", func() {
		err := CopyToFile(":::invalid", "test", "/tmp/test-imagecopy-dest")
		Expect(err).To(HaveOccurred())
	})

	It("LoadToStorage fails with invalid image name", func() {
		err := LoadToStorage("/tmp/nonexistent", ":::invalid", "amd64")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid image name"))
	})

	It("LoadToStorage fails with nonexistent directory", func() {
		err := LoadToStorage("/tmp/nonexistent-dir-imagecopy-test", "localhost/registry:latest", "amd64")
		Expect(err).To(HaveOccurred())
	})
})

func TestImageCopy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "imagecopy_test")
}
