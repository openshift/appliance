package oc

import (
	"os"
	"path/filepath"

	"github.com/cavaliergopher/grab/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/appliance/pkg/tools"
)

const (
	ocBinaryName = "oc"
)

// ClientConfig holds the parameters needed to acquire the oc binary.
type ClientConfig struct {
	tools.ToolConfig
}

// Client handles download, extraction, and storage of the oc binary.
type Client struct {
	config ClientConfig
}

// NewClient creates a Client for acquiring the oc binary.
func NewClient(config ClientConfig) *Client {
	return &Client{config: config}
}

// AcquireResult holds the outcome of an Acquire call.
type AcquireResult struct {
	Path       string // absolute path to the oc binary
	FromSystem bool   // true when oc was found on PATH rather than downloaded
}

// Acquire returns the path to the oc binary. It first reuses an existing binary
// in the tools dir, then tries to download from mirror.openshift.com, and finally
// falls back to a system-installed oc on PATH.
func (c *Client) Acquire() (*AcquireResult, error) {
	existing := tools.FindExistingBinary(c.config.ToolsDir, ocBinaryName)
	if existing != "" {
		logrus.Infof("Reusing existing oc binary")
		return &AcquireResult{Path: existing}, nil
	}

	result, downloadErr := c.download()
	if downloadErr == nil {
		return result, nil
	}

	logrus.Warnf("Failed to download oc client: %v", downloadErr)
	systemPath, lookErr := tools.LookupSystemBinary(ocBinaryName)
	if lookErr != nil {
		return nil, errors.Wrap(downloadErr, "failed to download oc client and no system oc found on PATH")
	}

	logrus.Warnf("Using system oc at %s as fallback", systemPath)
	return &AcquireResult{Path: systemPath, FromSystem: true}, nil
}

func (c *Client) download() (*AcquireResult, error) {
	url, err := tools.GetDownloadURL(ocBinaryName, c.config.ReleaseVersion, c.config.SupportedArch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build oc download URL")
	}

	logrus.Infof("Downloading oc client from %s", url)
	archivePath := filepath.Join(c.config.ToolsDir, "openshift-client.tar.gz")
	_, err = grab.Get(archivePath, url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download oc client")
	}
	defer func() { _ = os.Remove(archivePath) }()

	ocPath, err := tools.ExtractBinaryFromArchive(archivePath, c.config.ToolsDir, ocBinaryName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract oc from archive")
	}

	if err := os.Chmod(ocPath, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to make oc executable")
	}

	logrus.Infof("oc client ready at %s", ocPath)
	return &AcquireResult{Path: ocPath}, nil
}
