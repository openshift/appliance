package imagecopy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/directory"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=imagecopy.go -package=imagecopy -destination=mock_imagecopy.go

// ImageCopier copies container images between transports.
type ImageCopier interface {
	// CopyToFile copies a container image from a Docker registry to a local
	// directory in OCI dir format, preserving all architectures and digests.
	CopyToFile(imageUrl, imageName, filePath string) error
}

type imageCopier struct{}

func NewImageCopier() ImageCopier {
	return &imageCopier{}
}

func (c *imageCopier) CopyToFile(imageUrl, imageName, filePath string) error {
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

// getPolicyContext creates a signature policy context.
// It tries the system default policy first, falling back to an
// accept-all policy if the system policy is not available (e.g. on macOS).
func getPolicyContext() (*signature.PolicyContext, error) {
	policy, err := signature.DefaultPolicy(nil)
	if err != nil {
		logrus.Debugf("System signature policy not found, using insecure accept-all policy: %v", err)
		policy = &signature.Policy{
			Default: []signature.PolicyRequirement{
				signature.NewPRInsecureAcceptAnything(),
			},
		}
	}
	return signature.NewPolicyContext(policy)
}
