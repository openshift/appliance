package releasebundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/openshift/appliance/pkg/executer"
	"github.com/pkg/errors"
)

const bundlePushCmd = "podman push --tls-verify=false %s"

type BundleConfig struct {
	Executer       executer.Executer
	Port           int
	ReleaseVersion string
	// ImageSetPath is the absolute path to the rendered imageset.yaml used for oc mirror.
	ImageSetPath string
	// MappingBytes is oc-mirror mapping.txt content (may be nil if not produced).
	MappingBytes []byte
}

type Bundle struct {
	BundleConfig
}

func NewBundle(config BundleConfig) *Bundle {
	if config.Executer == nil {
		config.Executer = executer.NewExecuter()
	}
	return &Bundle{BundleConfig: config}
}

func (b *Bundle) Push() error {
	if b.ImageSetPath == "" {
		return errors.New("bundle: ImageSetPath is required")
	}
	dockerfileSrc, err := readBundleDockerfile()
	if err != nil {
		return err
	}

	stagedir, err := os.MkdirTemp("", "appliance-release-bundle-*")
	if err != nil {
		return errors.Wrap(err, "create bundle staging dir")
	}
	defer os.RemoveAll(stagedir)

	dockerfileDest := filepath.Join(stagedir, "Dockerfile.bundle")
	if err := os.WriteFile(dockerfileDest, dockerfileSrc, 0o644); err != nil {
		return errors.Wrap(err, "write staged Dockerfile.bundle")
	}

	imageSetSrc, err := os.ReadFile(b.ImageSetPath)
	if err != nil {
		return errors.Wrap(err, "read imageset for bundle")
	}
	if err := os.WriteFile(filepath.Join(stagedir, "imageset.yaml"), imageSetSrc, 0o644); err != nil {
		return errors.Wrap(err, "write staged imageset.yaml")
	}

	mapping := b.MappingBytes
	if len(mapping) == 0 {
		mapping = []byte("# mapping.txt was not found under the oc-mirror workspace\n")
	}
	if err := os.WriteFile(filepath.Join(stagedir, "mapping.txt"), mapping, 0o644); err != nil {
		return errors.Wrap(err, "write staged mapping.txt")
	}

	tag := Tag(b.ReleaseVersion)
	imageRef := registryImageRef(b.Port, tag)
	bundleVer := b.ReleaseVersion
	if bundleVer == "" {
		bundleVer = "unknown"
	}
	buildCmd := fmt.Sprintf(
		"podman build --build-arg BUNDLE_VERSION=%s --build-arg BUNDLE_RELEASE=1 -f %s -t %s %s",
		strconv.Quote(bundleVer), dockerfileDest, imageRef, stagedir,
	)
	if _, err := b.Executer.Execute(buildCmd); err != nil {
		return errors.Wrap(err, "build release bundle image")
	}

	pushCmd := fmt.Sprintf(bundlePushCmd, imageRef)
	if _, err := b.Executer.Execute(pushCmd); err != nil {
		return errors.Wrap(err, "push release bundle image")
	}

	return nil
}

func readBundleDockerfile() ([]byte, error) {
	path, err := bundleDockerfileAbsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read bundle Dockerfile.bundle")
	}
	return data, nil
}

func bundleDockerfileAbsPath() (string, error) {
	path, _, err := resolveDockerfile()
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, path), nil
}

// registryImageRef is the image reference used for podman build/push against the local registry
// during appliance data ISO generation (must stay aligned with oc mirror localhost layout).
func registryImageRef(port int, tag string) string {
	return fmt.Sprintf("127.0.0.1:%d/%s:%s", port, ImageRepository, tag)
}

// resolveDockerfile returns paths for locating Dockerfile.bundle (path may be relative to cwd).
func resolveDockerfile() (dockerfilePath, contextDir string, err error) {
	candidates := []struct {
		dockerfile string
		context    string
	}{
		{"/Dockerfile.bundle", "/"},
		{filepath.Join("bundle", "Dockerfile.bundle"), "bundle"},
	}
	for _, c := range candidates {
		if st, statErr := os.Stat(c.dockerfile); statErr == nil && !st.IsDir() {
			return c.dockerfile, c.context, nil
		}
	}
	return "", "", errors.New("Dockerfile.bundle not found (expected /Dockerfile.bundle in the appliance image or bundle/Dockerfile.bundle when run from the repository root)")
}
