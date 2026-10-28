package ocmirror

import (
	"os"
	"path/filepath"

	"github.com/cavaliergopher/grab/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/appliance/pkg/tools"
)

const (
	ocMirrorBinaryName = "oc-mirror"
)

// MirrorConfig holds the parameters needed to acquire the oc-mirror binary.
type MirrorConfig struct {
	tools.ToolConfig
}

// Mirror handles download, extraction, and storage of the oc-mirror binary.
type Mirror struct {
	config MirrorConfig
}

// NewMirror creates a Mirror for acquiring the oc-mirror binary.
func NewMirror(config MirrorConfig) *Mirror {
	return &Mirror{config: config}
}

// AcquireResult holds the outcome of an Acquire call.
type AcquireResult struct {
	Path       string // absolute path to the oc-mirror binary
	FromSystem bool   // true when oc-mirror was found on PATH rather than downloaded
}

// Acquire returns the path to the oc-mirror binary. It first reuses an existing binary
// in the tools dir, then tries to download from mirror.openshift.com, and finally
// falls back to a system-installed oc-mirror on PATH.
func (m *Mirror) Acquire() (*AcquireResult, error) {
	existing := tools.FindExistingBinary(m.config.ToolsDir, ocMirrorBinaryName)
	if existing != "" {
		logrus.Infof("Reusing existing oc-mirror binary")
		return &AcquireResult{Path: existing}, nil
	}

	result, downloadErr := m.download()
	if downloadErr == nil {
		return result, nil
	}

	logrus.Warnf("Failed to download oc-mirror: %v", downloadErr)
	systemPath, lookErr := tools.LookupSystemBinary(ocMirrorBinaryName)
	if lookErr != nil {
		return nil, errors.Wrap(downloadErr, "failed to download oc-mirror and no system oc-mirror found on PATH")
	}

	logrus.Warnf("Using system oc-mirror at %s as fallback", systemPath)
	return &AcquireResult{Path: systemPath, FromSystem: true}, nil
}

func (m *Mirror) download() (*AcquireResult, error) {
	url, err := tools.GetDownloadURL(ocMirrorBinaryName, m.config.ReleaseVersion, m.config.SupportedArch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build oc-mirror download URL")
	}

	logrus.Infof("Downloading oc-mirror from %s", url)
	archivePath := filepath.Join(m.config.ToolsDir, "oc-mirror.tar.gz")
	_, err = grab.Get(archivePath, url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download oc-mirror")
	}
	defer func() { _ = os.Remove(archivePath) }()

	ocMirrorPath, err := tools.ExtractBinaryFromArchive(archivePath, m.config.ToolsDir, ocMirrorBinaryName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract oc-mirror from archive")
	}

	if err := os.Chmod(ocMirrorPath, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to make oc-mirror executable")
	}

	logrus.Infof("oc-mirror ready at %s", ocMirrorPath)
	return &AcquireResult{Path: ocMirrorPath}, nil
}
