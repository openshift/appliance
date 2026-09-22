package imagecopy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.podman.io/image/v5/copy"
	"go.podman.io/image/v5/directory"
	"go.podman.io/image/v5/docker"
	"go.podman.io/image/v5/docker/reference"
	"go.podman.io/image/v5/signature"
	imgStorage "go.podman.io/image/v5/storage"
	"go.podman.io/image/v5/types"
	"go.podman.io/storage"
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

// LoadToStorage copies a container image from a local directory in dir
// format into the containers-storage, tagging it with the given name.
// arch specifies the target platform architecture in OCI format (e.g.
// "amd64", "arm64", "ppc64le").
func LoadToStorage(dirPath, imageName, arch string) error {
	named, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return fmt.Errorf("invalid image name %q: %w", imageName, err)
	}

	storeOpts, err := storage.DefaultStoreOptions()
	if err != nil {
		return fmt.Errorf("failed to get default store options: %w", err)
	}

	store, err := storage.GetStore(storeOpts)
	if err != nil {
		return fmt.Errorf("failed to open containers storage: %w", err)
	}
	defer func() {
		if _, err := store.Shutdown(false); err != nil {
			logrus.Warnf("failed to shut down containers storage: %v", err)
		}
	}()

	destRef, err := imgStorage.Transport.NewStoreReference(store, named, "")
	if err != nil {
		return fmt.Errorf("failed to create storage reference for %q: %w", imageName, err)
	}

	srcRef, err := directory.NewReference(dirPath)
	if err != nil {
		return fmt.Errorf("invalid source directory %q: %w", dirPath, err)
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

	ctx := context.Background()
	logrus.Debugf("Loading image from %s to containers-storage as %s", dirPath, imageName)
	_, err = copy.Image(ctx, policyCtx, destRef, srcRef, &copy.Options{
		ImageListSelection: copy.CopySystemImage,
		SourceCtx:          &types.SystemContext{ArchitectureChoice: arch},
		DestinationCtx:     &types.SystemContext{},
	})
	if err != nil {
		return fmt.Errorf("failed to load image from %s to storage: %w", dirPath, err)
	}

	logrus.Debugf("Successfully loaded image %s into containers-storage", imageName)
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
