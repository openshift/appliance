package appliance

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/swag/conv"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-image-service/pkg/isoeditor"
)

// trackingReadCloser records whether Close was called, so tests can assert
// that writeKargFiles does not leak file descriptors.
type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}

var _ = Describe("fipsKargs", func() {
	DescribeTable("returns the right kernel arguments",
		func(enableFips *bool, expected string) {
			Expect(fipsKargs(enableFips)).To(Equal(expected))
		},
		Entry("nil pointer means not enabled", nil, ""),
		Entry("explicitly false", conv.Pointer(false), ""),
		// The leading space is required: isoeditor splices content over the
		// newline that terminates the "linux ..." line, so the appended
		// arguments must supply their own separator. Without it the result is
		// "...ignition.platform.id=metalfips=1".
		Entry("explicitly true", conv.Pointer(true), " fips=1"),
	)
})

var _ = Describe("writeKargFiles", func() {
	var workDir string

	BeforeEach(func() {
		var err error
		workDir, err = os.MkdirTemp("", "kargs-test")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			Expect(os.RemoveAll(workDir)).To(Succeed())
		})
	})

	It("overwrites the target file with the new content", func() {
		Expect(os.MkdirAll(filepath.Join(workDir, "EFI/redhat"), 0o755)).To(Succeed())
		target := filepath.Join(workDir, "EFI/redhat/grub.cfg")
		Expect(os.WriteFile(target, []byte("original"), 0o644)).To(Succeed())

		files := []isoeditor.FileData{{
			Filename: "EFI/redhat/grub.cfg",
			Data:     &trackingReadCloser{Reader: strings.NewReader("patched fips=1")},
		}}

		Expect(writeKargFiles(workDir, files)).To(Succeed())

		content, err := os.ReadFile(target)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("patched fips=1"))
	})

	It("writes every file when there is more than one boot path", func() {
		Expect(os.MkdirAll(filepath.Join(workDir, "EFI/redhat"), 0o755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(workDir, "isolinux"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workDir, "EFI/redhat/grub.cfg"), []byte("x"), 0o644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workDir, "isolinux/isolinux.cfg"), []byte("x"), 0o644)).To(Succeed())

		files := []isoeditor.FileData{
			{Filename: "EFI/redhat/grub.cfg", Data: &trackingReadCloser{Reader: strings.NewReader("uefi")}},
			{Filename: "isolinux/isolinux.cfg", Data: &trackingReadCloser{Reader: strings.NewReader("bios")}},
		}

		Expect(writeKargFiles(workDir, files)).To(Succeed())

		uefi, err := os.ReadFile(filepath.Join(workDir, "EFI/redhat/grub.cfg"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(uefi)).To(Equal("uefi"))

		bios, err := os.ReadFile(filepath.Join(workDir, "isolinux/isolinux.cfg"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(bios)).To(Equal("bios"))
	})

	It("closes every reader on success", func() {
		Expect(os.MkdirAll(filepath.Join(workDir, "EFI/redhat"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workDir, "EFI/redhat/grub.cfg"), []byte("x"), 0o644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workDir, "EFI/redhat/other.cfg"), []byte("x"), 0o644)).To(Succeed())
		first := &trackingReadCloser{Reader: strings.NewReader("a")}
		second := &trackingReadCloser{Reader: strings.NewReader("b")}

		files := []isoeditor.FileData{
			{Filename: "EFI/redhat/grub.cfg", Data: first},
			{Filename: "EFI/redhat/other.cfg", Data: second},
		}

		Expect(writeKargFiles(workDir, files)).To(Succeed())
		Expect(first.closed).To(BeTrue())
		Expect(second.closed).To(BeTrue())
	})

	It("fails instead of creating a stray file when the boot file is absent", func() {
		// The directory exists but the boot file does not. If the filename
		// from kargs.json ever disagrees with what Extract produced, creating
		// a new file here would leave the real boot config unpatched and the
		// build would report success — a silently non-FIPS ISO.
		Expect(os.MkdirAll(filepath.Join(workDir, "EFI/redhat"), 0o755)).To(Succeed())
		absent := filepath.Join(workDir, "EFI/redhat/grub.cfg")

		files := []isoeditor.FileData{{
			Filename: "/EFI/redhat/grub.cfg",
			Data:     &trackingReadCloser{Reader: strings.NewReader("patched")},
		}}

		err := writeKargFiles(workDir, files)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("EFI/redhat/grub.cfg"))
		Expect(absent).NotTo(BeAnExistingFile())
	})

	It("closes remaining readers even when a write fails", func() {
		// EFI/redhat is deliberately not created, so the first write fails.
		failing := &trackingReadCloser{Reader: strings.NewReader("a")}
		untouched := &trackingReadCloser{Reader: strings.NewReader("b")}

		files := []isoeditor.FileData{
			{Filename: "EFI/redhat/grub.cfg", Data: failing},
			{Filename: "isolinux/isolinux.cfg", Data: untouched},
		}

		err := writeKargFiles(workDir, files)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("EFI/redhat/grub.cfg"))
		Expect(failing.closed).To(BeTrue())
		Expect(untouched.closed).To(BeTrue())
	})
})

var _ = Describe("appendFipsKargs", func() {
	var (
		workDir        string
		originalReader func(string, string) ([]isoeditor.FileData, error)
	)

	BeforeEach(func() {
		var err error
		workDir, err = os.MkdirTemp("", "appendkargs-test")
		Expect(err).NotTo(HaveOccurred())
		originalReader = kargsReader
		DeferCleanup(func() {
			kargsReader = originalReader
			Expect(os.RemoveAll(workDir)).To(Succeed())
		})
	})

	It("does nothing when FIPS is not enabled", func() {
		called := false
		kargsReader = func(isoPath, kargs string) ([]isoeditor.FileData, error) {
			called = true
			return nil, nil
		}

		Expect(appendFipsKargs("/nonexistent.iso", workDir, conv.Pointer(false))).To(Succeed())
		Expect(called).To(BeFalse())
	})

	It("does nothing when enableFips is nil", func() {
		called := false
		kargsReader = func(isoPath, kargs string) ([]isoeditor.FileData, error) {
			called = true
			return nil, nil
		}

		Expect(appendFipsKargs("/nonexistent.iso", workDir, nil)).To(Succeed())
		Expect(called).To(BeFalse())
	})

	It("requests exactly the fips=1 argument when enabled", func() {
		var gotIsoPath, gotKargs string
		Expect(os.MkdirAll(filepath.Join(workDir, "EFI/redhat"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(workDir, "EFI/redhat/grub.cfg"), []byte("x"), 0o644)).To(Succeed())
		kargsReader = func(isoPath, kargs string) ([]isoeditor.FileData, error) {
			gotIsoPath, gotKargs = isoPath, kargs
			return []isoeditor.FileData{{
				Filename: "EFI/redhat/grub.cfg",
				Data:     &trackingReadCloser{Reader: strings.NewReader("patched")},
			}}, nil
		}

		Expect(appendFipsKargs("/base.iso", workDir, conv.Pointer(true))).To(Succeed())
		Expect(gotIsoPath).To(Equal("/base.iso"))
		Expect(gotKargs).To(Equal(" fips=1"))

		content, err := os.ReadFile(filepath.Join(workDir, "EFI/redhat/grub.cfg"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("patched"))
	})

	It("fails loudly when the ISO has no kernel argument embed area", func() {
		// isoeditor returns an empty slice and no error for an ISO without
		// /coreos/kargs.json. Silently succeeding here would ship a non-FIPS
		// ISO to a user who asked for FIPS.
		kargsReader = func(isoPath, kargs string) ([]isoeditor.FileData, error) {
			return nil, nil
		}

		err := appendFipsKargs("/base.iso", workDir, conv.Pointer(true))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no boot configuration files"))
	})

	It("propagates an error from the kargs reader", func() {
		kargsReader = func(isoPath, kargs string) ([]isoeditor.FileData, error) {
			return nil, fmt.Errorf("boom")
		}

		err := appendFipsKargs("/base.iso", workDir, conv.Pointer(true))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("boom"))
	})
})

func TestAppliance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "appliance_test")
}
