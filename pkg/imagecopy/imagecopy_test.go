package imagecopy

import (
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test ImageCopier", func() {
	It("NewImageCopier returns non-nil instance", func() {
		copier := NewImageCopier()
		Expect(copier).ToNot(BeNil())
	})

	It("CopyToFile fails with invalid image URL", func() {
		copier := NewImageCopier()
		err := copier.CopyToFile(":::invalid", "test", "/tmp/test-imagecopy-dest")
		Expect(err).To(HaveOccurred())
	})
})

func TestImageCopy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "imagecopy_test")
}
