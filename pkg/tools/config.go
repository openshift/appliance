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

const (
	templateDownloadURL = "https://mirror.openshift.com/pub/openshift-v%s/%s/clients/%s/%s/%s"
	// ocMirrorDownloadURL is the location of the latest oc-mirror release.
	ocMirrorDownloadURL = "https://mirror.openshift.com/pub/cgw/oc-mirror/latest/%s"
)

// Identifiers of the host platforms supported for tool acquisition.
const (
	platformLinuxAmd64  = "linux-amd64"
	platformDarwinAmd64 = "darwin-amd64"
	platformDarwinArm64 = "darwin-arm64"
)

// ToolConfig holds the common parameters needed to acquire external tools.
type ToolConfig struct {
	ReleaseVersion string // resolved OCP version, e.g. "4.22.19"
	ToolsDir       string // directory for storing the binary
	SupportedArch  string // mirror.openshift.com arch path segment, e.g. "x86_64"
}

// ocMirrorFiles maps the platforms supported by oc-mirror to its archive filenames.
var ocMirrorFiles = map[string]string{
	platformLinuxAmd64: "oc-mirror-rhel9-" + platformLinuxAmd64 + ".tar.gz",
}

// tools maps tool names to their archive filenames by platform.
var tools = map[string]map[string]string{
	"oc": {
		platformLinuxAmd64:  "openshift-client-linux.tar.gz",
		platformDarwinAmd64: "openshift-client-mac.tar.gz",
		platformDarwinArm64: "openshift-client-mac-arm64.tar.gz",
	},
}

// getSupportedPlatform returns a platform identifier for the host system.
// It validates that the platform is supported for tool acquisition.
func getSupportedPlatform() (string, error) {
	switch {
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return platformLinuxAmd64, nil
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		return platformDarwinAmd64, nil
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return platformDarwinArm64, nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

// getToolName returns the archive filename for the specified tool on the given platform.
func getToolName(toolName, platformArch string) (string, error) {
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

// GetDownloadURL returns the download URL for the specified tool on the host platform.
// oc-mirror is backward compatible with older OpenShift versions, so its latest release
// is always used.
func GetDownloadURL(toolName string, releaseVersion string, supportedArch string) (string, error) {
	platform, err := getSupportedPlatform()
	if err != nil {
		return "", err
	}
	return getDownloadURL(toolName, releaseVersion, supportedArch, platform)
}

func getDownloadURL(toolName, releaseVersion, supportedArch, platform string) (string, error) {
	if toolName == "oc-mirror" {
		file, ok := ocMirrorFiles[platform]
		if !ok {
			return "", errors.Errorf("oc-mirror is not published for %s, provide it on PATH (it can be built from https://github.com/openshift/oc-mirror)", platform)
		}
		return fmt.Sprintf(ocMirrorDownloadURL, file), nil
	}

	filename, err := getToolName(toolName, platform)
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
