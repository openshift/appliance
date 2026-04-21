package skopeo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/executer"
)

const (
	// templateCopyToFile copies container images to dir: format with critical flags:
	//
	// --all: Copies all architectures/platforms including the manifest list
	//        This is required to reference the image by its original digest because:
	//        1. Multi-arch images have a top-level manifest list with a digest (e.g., sha256:abc123...)
	//        2. REGISTRY_IMAGE may reference this manifest list digest
	//           (e.g., registry.ci.openshift.org/ocp/4.21@sha256:abc123...)
	//        3. Without --all, only a single architecture manifest is copied, losing the original
	//           manifest list digest, making the image unreferenceable by the original digest
	//        4. start-local-registry.service needs to podman run using this original digest
	//
	// --preserve-digests: Preserves architecture-specific manifest digests
	//        Each architecture (amd64, arm64, ppc64le) has its own manifest with a unique digest
	//        This flag ensures these individual architecture digests are maintained in the dir: format
	//
	// dir: format: Stores the image as a directory structure instead of a tar archive
	//        This format preserves all image metadata and supports podman pull dir: for loading
	templateCopyToFile = "skopeo copy --all --preserve-digests docker://%s dir:%s"
)

type Skopeo interface {
	CopyToFile(imageUrl, imageName, filePath string) error
}

type skopeo struct {
	executer executer.Executer
}

func NewSkopeo(exec executer.Executer) Skopeo {
	if exec == nil {
		exec = executer.NewExecuter()
	}

	return &skopeo{
		executer: exec,
	}
}

func (s *skopeo) CopyToFile(imageUrl, imageName, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	_, err := s.executer.Execute(fmt.Sprintf(templateCopyToFile, imageUrl, filePath))
	return err
}
