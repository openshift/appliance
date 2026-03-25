package releasebundle

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/executer"
	"github.com/pkg/errors"
)

const (
	bundleBuildCmd = "podman build -f %s -t %s %s"
	bundlePushCmd  = "podman push --tls-verify=false %s"
)

type BundleConfig struct {
	Executer       executer.Executer
	Port           int
	ReleaseVersion string
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
	dockerfilePath, ctx, err := resolveDockerfile()
	if err != nil {
		return err
	}

	tag := Tag(b.ReleaseVersion)
	imageRef := registryImageRef(b.Port, tag)
	buildCmd := fmt.Sprintf(bundleBuildCmd, dockerfilePath, imageRef, ctx)
	if _, err := b.Executer.Execute(buildCmd); err != nil {
		return errors.Wrap(err, "build release bundle image")
	}

	pushCmd := fmt.Sprintf(bundlePushCmd, imageRef)
	if _, err := b.Executer.Execute(pushCmd); err != nil {
		return errors.Wrap(err, "push release bundle image")
	}

	return nil
}

// registryImageRef is the image reference used for podman build/push against the local registry
// during appliance data ISO generation (must stay aligned with oc mirror localhost layout).
func registryImageRef(port int, tag string) string {
	return fmt.Sprintf("127.0.0.1:%d/%s:%s", port, ImageRepository, tag)
}

// resolveDockerfile returns paths for podman build: Dockerfile path and build context directory.
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
