package releasebundle

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/fileutil"
)

const (
	bundleBuildCmd = "podman build -f %s -t %s %s"
	bundlePushCmd  = "podman push --tls-verify=false %s"

	// ManifestsDir is the directory in the bundle image holding the oc-mirror manifests.
	ManifestsDir = "manifests"

	// fallbackDockerfile must match bundle/Dockerfile.bundle.
	fallbackDockerfile = "FROM scratch\nCOPY " + ManifestsDir + "/ /" + ManifestsDir + "/\n"
)

// BundleConfig holds the settings used to build and push the release bundle image.
type BundleConfig struct {
	Executer       executer.Executer
	Port           int
	ReleaseVersion string
	// TempDir is where the build context is staged; defaults to the OS temp dir.
	TempDir string
	// Manifests are files or directories copied into the image under /manifests.
	// Paths that do not exist are skipped.
	Manifests []string
}

// Bundle builds and pushes the release bundle image to the local registry.
type Bundle struct {
	BundleConfig
}

// NewBundle returns a Bundle for the given config.
func NewBundle(config BundleConfig) *Bundle {
	if config.Executer == nil {
		config.Executer = executer.NewExecuter()
	}
	return &Bundle{BundleConfig: config}
}

// Push builds the release bundle image, including the configured manifests,
// and pushes it to the local registry.
func (b *Bundle) Push() error {
	contextDir, err := os.MkdirTemp(b.TempDir, "release-bundle-")
	if err != nil {
		return errors.Wrap(err, "create release bundle build context")
	}
	defer func() {
		if err := os.RemoveAll(contextDir); err != nil {
			logrus.Warnf("Failed to remove release bundle build context %s: %v", contextDir, err)
		}
	}()

	if err = b.stageManifests(filepath.Join(contextDir, ManifestsDir)); err != nil {
		return errors.Wrap(err, "stage release bundle manifests")
	}

	imageRef := registryImageRef(b.Port, Tag(b.ReleaseVersion))

	// When running inside the appliance container, Dockerfile.bundle is at
	// /Dockerfile.bundle; from the repo root it's at bundle/Dockerfile.bundle.
	// For standalone iso-builder neither path exists, so we fall back to piping
	// the equivalent Dockerfile via stdin to avoid any file dependency.
	// The fallback uses exec.Command directly because the Executer splits on
	// spaces and cannot handle shell pipes.
	if dockerfilePath, err := resolveDockerfile(); err == nil {
		buildCmd := fmt.Sprintf(bundleBuildCmd, dockerfilePath, imageRef, contextDir)
		if _, err := b.Executer.Execute(buildCmd); err != nil {
			return errors.Wrap(err, "build release bundle image")
		}
	} else {
		pipeline := fmt.Sprintf("printf '%%s' '%s' | podman build -f - -t %s %s", fallbackDockerfile, imageRef, contextDir)
		if out, cmdErr := exec.Command("bash", "-c", pipeline).CombinedOutput(); cmdErr != nil {
			return errors.Wrap(fmt.Errorf("%s: %s", cmdErr, string(out)), "build release bundle image")
		}
	}

	pushCmd := fmt.Sprintf(bundlePushCmd, imageRef)
	if _, err := b.Executer.Execute(pushCmd); err != nil {
		return errors.Wrap(err, "push release bundle image")
	}

	return nil
}

// stageManifests copies each configured manifest path into destDir, keeping its base name.
func (b *Bundle) stageManifests(destDir string) error {
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return err
	}
	for _, src := range b.Manifests {
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				logrus.Debugf("Release bundle manifest not found, skipping: %s", src)
				continue
			}
			return err
		}
		if err := copyTree(src, filepath.Join(destDir, filepath.Base(src))); err != nil {
			return errors.Wrapf(err, "copy %s", src)
		}
	}
	return nil
}

// copyTree copies a file, or the regular files of a directory tree, from src to dest.
func copyTree(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		return fileutil.CopyFile(path, filepath.Join(dest, rel))
	})
}

// registryImageRef is the image reference used for podman build/push against the local registry
// during appliance data ISO generation (must stay aligned with oc mirror localhost layout).
func registryImageRef(port int, tag string) string {
	return fmt.Sprintf("127.0.0.1:%d/%s:%s", port, ImageRepository, tag)
}

// resolveDockerfile returns the path of Dockerfile.bundle used for podman build.
func resolveDockerfile() (string, error) {
	candidates := []string{
		"/Dockerfile.bundle",
		filepath.Join("bundle", "Dockerfile.bundle"),
	}
	for _, c := range candidates {
		if st, statErr := os.Stat(c); statErr == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", errors.New("Dockerfile.bundle not found (expected /Dockerfile.bundle in the appliance image or bundle/Dockerfile.bundle when run from the repository root)")
}
