package tools

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

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

const templateDownloadURL = "https://mirror.openshift.com/pub/openshift-v%s/%s/clients/%s/%s/%s"

// ToolConfig holds the common parameters needed to acquire external tools.
type ToolConfig struct {
	ReleaseVersion string // resolved OCP version, e.g. "4.22.19"
	ToolsDir       string // directory for storing the binary
	SupportedArch  string // mirror.openshift.com arch path segment, e.g. "x86_64"
}

// tools maps tool names to their archive filenames by platform.
var tools = map[string]map[string]string{
	"oc": {
		"linux-amd64":   "openshift-client-linux.tar.gz",
		"darwin-amd64":  "openshift-client-mac.tar.gz",
		"darwin-arm64":  "openshift-client-mac-arm64.tar.gz",
	},
	"oc-mirror": {
		"linux-amd64":   "oc-mirror.tar.gz",
		"darwin-amd64":  "oc-mirror-mac.tar.gz",
		"darwin-arm64":  "oc-mirror-mac-arm64.tar.gz",
	},
}

// getSupportedPlatform returns a platform identifier for the host system.
// It validates that the platform is supported for tool acquisition.
func getSupportedPlatform() (string, error) {
	switch {
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "linux-amd64", nil
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		return "darwin-amd64", nil
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return "darwin-arm64", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

// getToolName returns the archive filename for the specified tool on the host platform.
func getToolName(toolName string) (string, error) {
	platformArch, err := getSupportedPlatform()
	if err != nil {
		return "", err
	}

	tools, ok := tools[toolName]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}

	tool, ok := tools[platformArch]
	if !ok {
		return "", fmt.Errorf("unsupported platform: %s", platformArch)
	}

	return tool, nil
}

// GetDownloadURL returns the download URL for the specified tool.
func GetDownloadURL(toolName string, releaseVersion string, supportedArch string) (string, error) {
	filename, err := getToolName(toolName)
	if err != nil {
		return "", err
	}

	v, err := version.NewVersion(releaseVersion)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse release version")
	}
	majorVersion := fmt.Sprint(v.Segments()[0])

	ocpClient := "ocp"
	if strings.Contains(releaseVersion, "-ec") {
		ocpClient = "ocp-dev-preview"
	}

	return fmt.Sprintf(templateDownloadURL, majorVersion, supportedArch, ocpClient, releaseVersion, filename), nil
}

// CreateToolsDir creates the tools directory if it doesn't exist.
func CreateToolsDir(toolsDir string) error {
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create tools dir")
	}
	return nil
}

// FindExistingBinary checks if a binary exists in the given directory.
func FindExistingBinary(toolsDir string, binaryName string) string {
	path := filepath.Join(toolsDir, binaryName)
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return ""
	}
	return path
}

// ExtractBinaryFromArchive extracts a binary from a tar.gz archive.
func ExtractBinaryFromArchive(archivePath, destDir, binaryName string) (string, error) {
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
		if filepath.Base(hdr.Name) == binaryName && hdr.Typeflag == tar.TypeReg {
			destPath := filepath.Join(destDir, binaryName)
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
	return "", fmt.Errorf("%s binary not found in archive", binaryName)
}

// LookupSystemBinary searches for a binary in the system PATH.
func LookupSystemBinary(binaryName string) (string, error) {
	return exec.LookPath(binaryName)
}
