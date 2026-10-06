package oc

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cavaliergopher/grab/v3"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	goOS     = runtime.GOOS
	goArch   = runtime.GOARCH
	lookPath = exec.LookPath
)

const (
	ocBinaryName        = "oc"
	templateDownloadURL = "https://mirror.openshift.com/pub/openshift-v%s/%s/clients/%s/%s/%s"
)

// ClientConfig holds the parameters needed to acquire the oc binary.
type ClientConfig struct {
	ReleaseVersion string // resolved OCP version, e.g. "4.22.3"
	CacheDir       string // directory for storing the binary
}

// Client handles download, extraction, and caching of the oc binary.
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

// Acquire returns the path to the oc binary. It first checks the cache, then
// tries to download from mirror.openshift.com, and finally falls back to a
// system-installed oc on PATH.
func (c *Client) Acquire() (*AcquireResult, error) {
	cached := findInCache(c.config.CacheDir, ocBinaryName)
	if cached != "" {
		logrus.Infof("Reusing oc binary from cache")
		return &AcquireResult{Path: cached}, nil
	}

	result, downloadErr := c.download()
	if downloadErr == nil {
		return result, nil
	}

	logrus.Warnf("Failed to download oc client: %v", downloadErr)
	systemPath, lookErr := lookPath(ocBinaryName)
	if lookErr != nil {
		return nil, errors.Wrap(downloadErr, "failed to download oc client and no system oc found on PATH")
	}

	logrus.Warnf("Using system oc at %s as fallback", systemPath)
	return &AcquireResult{Path: systemPath, FromSystem: true}, nil
}

func (c *Client) download() (*AcquireResult, error) {
	url, err := c.downloadURL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build oc download URL")
	}

	logrus.Infof("Downloading oc client from %s", url)
	archivePath := filepath.Join(c.config.CacheDir, "openshift-client.tar.gz")
	_, err = grab.Get(archivePath, url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download oc client")
	}
	defer func() { _ = os.Remove(archivePath) }()

	ocPath, err := extractOcFromArchive(archivePath, c.config.CacheDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract oc from archive")
	}

	if err := os.Chmod(ocPath, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to make oc executable")
	}

	logrus.Infof("oc client ready at %s", ocPath)
	return &AcquireResult{Path: ocPath}, nil
}

func (c *Client) downloadURL() (string, error) {
	v, err := version.NewVersion(c.config.ReleaseVersion)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse release version")
	}
	majorVersion := fmt.Sprint(v.Segments()[0])

	hostArch, err := mirrorArch()
	if err != nil {
		return "", err
	}

	ocpClient := "ocp"
	if strings.Contains(c.config.ReleaseVersion, "-ec") {
		ocpClient = "ocp-dev-preview"
	}

	filename, err := archiveFilename()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(templateDownloadURL, majorVersion, hostArch, ocpClient, c.config.ReleaseVersion, filename), nil
}

func mirrorArch() (string, error) {
	switch goArch {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", goArch)
	}
}

func archiveFilename() (string, error) {
	switch {
	case goOS == "linux" && goArch == "amd64":
		return "openshift-client-linux.tar.gz", nil
	case goOS == "linux" && goArch == "arm64":
		return "openshift-client-linux-arm64.tar.gz", nil
	case goOS == "darwin" && goArch == "amd64":
		return "openshift-client-mac.tar.gz", nil
	case goOS == "darwin" && goArch == "arm64":
		return "openshift-client-mac-arm64.tar.gz", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", goOS, goArch)
	}
}

func findInCache(cacheDir, name string) string {
	path := filepath.Join(cacheDir, name)
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return ""
	}
	return path
}

func extractOcFromArchive(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(hdr.Name) == ocBinaryName && hdr.Typeflag == tar.TypeReg {
			destPath := filepath.Join(destDir, ocBinaryName)
			out, err := os.Create(destPath)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil { // #nosec G110
				_ = out.Close()
				return "", err
			}
			if err := out.Close(); err != nil {
				return "", err
			}
			return destPath, nil
		}
	}
	return "", errors.New("oc binary not found in archive")
}
