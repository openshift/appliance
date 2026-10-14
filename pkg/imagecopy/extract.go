package imagecopy

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/pkg/compression"
	"go.podman.io/image/v5/types"

	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
)

const whiteoutPrefix = ".wh."

// ImageExtractor extracts files from a container image stored in dir format.
type ImageExtractor interface {
	ExtractFile(filePath, outputPath string) error
}

type imageExtractor struct {
	dirPath string
	arch    string
}

// NewImageExtractor creates an ImageExtractor for the dir-format image at
// dirPath. arch selects the platform for multi-arch images (e.g. "amd64").
func NewImageExtractor(dirPath, arch string) ImageExtractor {
	return &imageExtractor{dirPath: dirPath, arch: arch}
}

// ExtractFile extracts a single file from the container image layers
// and writes it to outputPath. filePath is the absolute path inside
// the container (e.g. "/usr/bin/distribution").
func (e *imageExtractor) ExtractFile(filePath, outputPath string) error {
	manifestBlob, err := os.ReadFile(filepath.Join(e.dirPath, "manifest.json"))
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	mimeType := manifest.GuessMIMEType(manifestBlob)

	if manifest.MIMETypeIsMultiImage(mimeType) {
		manifestBlob, err = e.resolveMultiArchManifest(manifestBlob, mimeType)
		if err != nil {
			return err
		}
		mimeType = manifest.GuessMIMEType(manifestBlob)
	}

	m, err := manifest.FromBlob(manifestBlob, mimeType)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	layers := m.LayerInfos()
	cleanPath := e.cleanFilePath(filePath)

	for i := len(layers) - 1; i >= 0; i-- {
		found, err := e.extractFromLayer(layers[i].Digest, cleanPath, outputPath)
		if err != nil {
			return fmt.Errorf("reading layer %s: %w", layers[i].Digest, err)
		}
		if found {
			logrus.Debugf("Extracted %s from layer %d (%s)", filePath, i, layers[i].Digest.Encoded()[:12])
			return nil
		}
	}

	return fmt.Errorf("file %q not found in image at %s", filePath, e.dirPath)
}

func (e *imageExtractor) resolveMultiArchManifest(manifestBlob []byte, mimeType string) ([]byte, error) {
	list, err := manifest.ListFromBlob(manifestBlob, mimeType)
	if err != nil {
		return nil, fmt.Errorf("parsing manifest list: %w", err)
	}

	instanceDigest, err := list.ChooseInstance(&types.SystemContext{
		ArchitectureChoice: e.arch,
	})
	if err != nil {
		return nil, fmt.Errorf("choosing instance for arch %s: %w", e.arch, err)
	}

	instancePath := filepath.Join(e.dirPath, instanceDigest.Encoded()+".manifest.json")
	instanceBlob, err := os.ReadFile(instancePath)
	if err != nil {
		return nil, fmt.Errorf("reading platform manifest %s: %w", instanceDigest, err)
	}

	return instanceBlob, nil
}

func (e *imageExtractor) extractFromLayer(layerDigest digest.Digest, targetPath, outputPath string) (bool, error) {
	blobPath := filepath.Join(e.dirPath, layerDigest.Encoded())
	f, err := os.Open(blobPath)
	if err != nil {
		return false, fmt.Errorf("opening blob %s: %w", layerDigest.Encoded()[:12], err)
	}
	defer f.Close()

	decompressed, _, err := compression.AutoDecompress(f)
	if err != nil {
		return false, fmt.Errorf("decompressing blob: %w", err)
	}
	defer decompressed.Close()

	tr := tar.NewReader(decompressed)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, fmt.Errorf("reading tar: %w", err)
		}

		clean := e.cleanFilePath(hdr.Name)

		if e.isWhiteout(clean, targetPath) {
			return false, fmt.Errorf("file %q was deleted (whiteout) in this layer", targetPath)
		}

		if clean != targetPath {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeReg:
			return true, e.writeFile(outputPath, tr, hdr.FileInfo().Mode())
		case tar.TypeSymlink, tar.TypeLink:
			return false, fmt.Errorf("file %q is a link (%s), extraction not supported", targetPath, hdr.Linkname)
		default:
			return false, fmt.Errorf("file %q has unexpected type %d", targetPath, hdr.Typeflag)
		}
	}

	return false, nil
}

func (e *imageExtractor) writeFile(outputPath string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	if mode&0o111 == 0 {
		mode |= 0o755
	}

	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, r); err != nil {
		return err
	}

	return nil
}

func (e *imageExtractor) cleanFilePath(p string) string {
	p = filepath.Clean(p)
	p = strings.TrimPrefix(p, ".")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func (e *imageExtractor) isWhiteout(entryPath, targetPath string) bool {
	dir := filepath.Dir(entryPath)
	base := filepath.Base(entryPath)
	if !strings.HasPrefix(base, whiteoutPrefix) {
		return false
	}
	originalName := strings.TrimPrefix(base, whiteoutPrefix)
	return filepath.Join(dir, originalName) == targetPath
}
