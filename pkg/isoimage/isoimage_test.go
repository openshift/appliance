package isoimage

import (
	"os"
	"path/filepath"
	"testing"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func readFileFromISO(isoPath, filePath string) ([]byte, error) {
	d, err := diskfs.Open(isoPath, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		return nil, err
	}
	fs, err := iso9660.Read(d.Backend, d.Size, 0, 0)
	if err != nil {
		return nil, err
	}
	f, err := fs.OpenFile(filePath, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	return buf[:n], nil
}

var _ = Describe("Test isoimage", func() {
	var (
		tmpDir  string
		outPath string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "isoimage-test-*")
		Expect(err).ToNot(HaveOccurred())
		outPath = filepath.Join(tmpDir, "test.iso")
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
	})

	It("creates an ISO with files at depth <= 8", func() {
		srcDir := filepath.Join(tmpDir, "src")
		dir := filepath.Join(srcDir, "a", "b", "c")
		Expect(os.MkdirAll(dir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)).To(Succeed())

		err := Create(outPath, srcDir, "TESTISO")
		Expect(err).ToNot(HaveOccurred())

		data, err := readFileFromISO(outPath, "/a/b/c/test.txt")
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal("hello"))
	})

	It("creates an ISO with directories deeper than 8 levels", func() {
		srcDir := filepath.Join(tmpDir, "src")
		// Simulate a Docker registry V2 path: 11 levels deep
		deepPath := filepath.Join(srcDir,
			"docker", "registry", "v2", "repositories",
			"ns", "name", "_manifests", "revisions",
			"sha256", "abc123")
		Expect(os.MkdirAll(deepPath, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(deepPath, "link"), []byte("sha256:abc123"), 0644)).To(Succeed())

		err := Create(outPath, srcDir, "DEEPISO")
		Expect(err).ToNot(HaveOccurred())

		data, err := readFileFromISO(outPath,
			"/docker/registry/v2/repositories/ns/name/_manifests/revisions/sha256/abc123/link")
		Expect(err).ToNot(HaveOccurred())
		Expect(string(data)).To(Equal("sha256:abc123"))
	})

	It("fails when workDir does not exist", func() {
		err := Create(outPath, filepath.Join(tmpDir, "nonexistent"), "FAIL")
		Expect(err).To(HaveOccurred())
	})
})

func TestIsoImage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "isoimage_test")
}
