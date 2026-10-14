package imagecopy

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

// buildTestDirImage creates a minimal dir-format container image with a single
// layer containing the given files. Returns the directory path.
func buildTestDirImage(tmpDir string, files map[string][]byte) string {
	dirPath := filepath.Join(tmpDir, "image")
	Expect(os.MkdirAll(dirPath, 0o755)).To(Succeed())

	layerDigest := writeLayer(dirPath, files)

	configContent := []byte(`{"architecture":"amd64","os":"linux"}`)
	configDigest := writeBlob(dirPath, configContent)

	manifest := map[string]interface{}{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2+json",
		"config": map[string]interface{}{
			"mediaType": "application/vnd.docker.container.image.v1+json",
			"digest":    configDigest,
			"size":      len(configContent),
		},
		"layers": []map[string]interface{}{
			{
				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"digest":    layerDigest,
			},
		},
	}

	manifestBytes, err := json.Marshal(manifest)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(dirPath, "manifest.json"), manifestBytes, 0o644)).To(Succeed())

	return dirPath
}

// buildTestDirImageMultiArch creates a dir-format manifest list with two
// platform manifests (amd64 and arm64), each with its own layer.
func buildTestDirImageMultiArch(tmpDir string, amd64Files, arm64Files map[string][]byte) string {
	dirPath := filepath.Join(tmpDir, "image")
	Expect(os.MkdirAll(dirPath, 0o755)).To(Succeed())

	amd64ManifestDigest := writePlatformManifest(dirPath, amd64Files)
	arm64ManifestDigest := writePlatformManifest(dirPath, arm64Files)

	index := map[string]interface{}{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.list.v2+json",
		"manifests": []map[string]interface{}{
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"digest":    amd64ManifestDigest,
				"platform":  map[string]string{"architecture": "amd64", "os": "linux"},
			},
			{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"digest":    arm64ManifestDigest,
				"platform":  map[string]string{"architecture": "arm64", "os": "linux"},
			},
		},
	}

	indexBytes, err := json.Marshal(index)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(dirPath, "manifest.json"), indexBytes, 0o644)).To(Succeed())

	return dirPath
}

func writePlatformManifest(dirPath string, files map[string][]byte) string {
	layerDigest := writeLayer(dirPath, files)

	configContent := []byte(`{"architecture":"amd64","os":"linux"}`)
	configDigest := writeBlob(dirPath, configContent)

	m := map[string]interface{}{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2+json",
		"config": map[string]interface{}{
			"mediaType": "application/vnd.docker.container.image.v1+json",
			"digest":    configDigest,
			"size":      len(configContent),
		},
		"layers": []map[string]interface{}{
			{
				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"digest":    layerDigest,
			},
		},
	}

	manifestBytes, err := json.Marshal(m)
	Expect(err).NotTo(HaveOccurred())

	h := sha256.Sum256(manifestBytes)
	digestStr := fmt.Sprintf("sha256:%x", h)
	encoded := fmt.Sprintf("%x", h)
	Expect(os.WriteFile(filepath.Join(dirPath, encoded+".manifest.json"), manifestBytes, 0o644)).To(Succeed())

	return digestStr
}

func writeLayer(dirPath string, files map[string][]byte) string {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		Expect(tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		})).To(Succeed())
		_, err := tw.Write(content)
		Expect(err).NotTo(HaveOccurred())
	}

	Expect(tw.Close()).To(Succeed())
	Expect(gw.Close()).To(Succeed())

	return writeBlob(dirPath, buf.Bytes())
}

func writeBlob(dirPath string, data []byte) string {
	h := sha256.Sum256(data)
	encoded := fmt.Sprintf("%x", h)
	Expect(os.WriteFile(filepath.Join(dirPath, encoded), data, 0o644)).To(Succeed())
	return fmt.Sprintf("sha256:%s", encoded)
}

var _ = Describe("ImageExtractor", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "extract-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("extracts a binary from a single-arch image", func() {
		content := []byte("#!/bin/sh\necho hello\n")
		dirPath := buildTestDirImage(tmpDir, map[string][]byte{
			"usr/bin/distribution": content,
		})
		outputPath := filepath.Join(tmpDir, "extracted-binary")

		extractor := NewImageExtractor(dirPath, "amd64")
		err := extractor.ExtractFile("/usr/bin/distribution", outputPath)
		Expect(err).NotTo(HaveOccurred())

		got, err := os.ReadFile(outputPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(content))

		info, err := os.Stat(outputPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode() & 0o111).NotTo(BeZero())
	})

	It("extracts from the correct platform in a multi-arch image", func() {
		amd64Content := []byte("amd64-binary")
		arm64Content := []byte("arm64-binary")

		dirPath := buildTestDirImageMultiArch(tmpDir,
			map[string][]byte{"usr/bin/distribution": amd64Content},
			map[string][]byte{"usr/bin/distribution": arm64Content},
		)
		outputPath := filepath.Join(tmpDir, "extracted-binary")

		extractor := NewImageExtractor(dirPath, "amd64")
		err := extractor.ExtractFile("/usr/bin/distribution", outputPath)
		Expect(err).NotTo(HaveOccurred())

		got, err := os.ReadFile(outputPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(amd64Content))
	})

	It("fails when the file is not in the image", func() {
		dirPath := buildTestDirImage(tmpDir, map[string][]byte{
			"usr/bin/other": []byte("other"),
		})
		outputPath := filepath.Join(tmpDir, "extracted-binary")

		extractor := NewImageExtractor(dirPath, "amd64")
		err := extractor.ExtractFile("/usr/bin/distribution", outputPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})

	It("fails when the image directory does not exist", func() {
		extractor := NewImageExtractor("/nonexistent-dir", "amd64")
		err := extractor.ExtractFile("/usr/bin/distribution", filepath.Join(tmpDir, "out"))
		Expect(err).To(HaveOccurred())
	})

	It("detects whiteout files", func() {
		dirPath := buildTestDirImage(tmpDir, map[string][]byte{
			"usr/bin/.wh.distribution": {},
		})
		outputPath := filepath.Join(tmpDir, "extracted-binary")

		extractor := NewImageExtractor(dirPath, "amd64")
		err := extractor.ExtractFile("/usr/bin/distribution", outputPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("whiteout"))
	})
})
