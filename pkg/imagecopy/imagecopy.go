package imagecopy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.podman.io/image/v5/copy"
	"go.podman.io/image/v5/directory"
	"go.podman.io/image/v5/docker"
	"go.podman.io/image/v5/signature"
	"go.podman.io/image/v5/types"

	"github.com/sirupsen/logrus"
)

// CopyToFile copies a container image from a Docker registry to a local
// directory in dir format, preserving all architectures and digests.
func CopyToFile(imageUrl, imageName, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	ctx := context.Background()

	srcRef, err := docker.ParseReference("//" + imageUrl)
	if err != nil {
		return fmt.Errorf("invalid source image reference %q: %w", imageUrl, err)
	}

	destRef, err := directory.NewReference(filePath)
	if err != nil {
		return fmt.Errorf("invalid destination path %q: %w", filePath, err)
	}

	policyCtx, err := getPolicyContext()
	if err != nil {
		return fmt.Errorf("failed to create policy context: %w", err)
	}
	defer func() {
		if err := policyCtx.Destroy(); err != nil {
			logrus.Warnf("failed to destroy policy context: %v", err)
		}
	}()

	logrus.Debugf("Copying image %s to %s", imageUrl, filePath)
	_, err = copy.Image(ctx, policyCtx, destRef, srcRef, &copy.Options{
		ImageListSelection: copy.CopyAllImages,
		PreserveDigests:    true,
		SourceCtx:          &types.SystemContext{},
		DestinationCtx:     &types.SystemContext{},
	})
	if err != nil {
		return fmt.Errorf("failed to copy image %s: %w", imageUrl, err)
	}

	logrus.Debugf("Successfully copied image %s to %s", imageUrl, filePath)
	return nil
}


// getPolicyContext creates a signature policy context with an accept-all
// policy. The tool copies known OCP images from trusted sources, so
// signature verification is not required.
func getPolicyContext() (*signature.PolicyContext, error) {
	policy := &signature.Policy{
		Default: []signature.PolicyRequirement{
			signature.NewPRInsecureAcceptAnything(),
		},
	}
	return signature.NewPolicyContext(policy)
}
