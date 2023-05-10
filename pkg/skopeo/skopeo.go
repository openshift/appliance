package skopeo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift/appliance/pkg/executer"
)

const (
	templateCopyToRegistry = "skopeo copy docker://%s docker://%s"
	templateCopyToFile     = "skopeo copy docker://%s docker-archive:%s:%s"
)

type Skopeo interface {
	CopyToRegistry(srcImage, dstImage string) error
	CopyToFile(imageUrl, imageName, filePath string) error
}

type skopeo struct {
	executer executer.Executer
}

func NewSkopeo() Skopeo {
	return &skopeo{
		executer: executer.NewExecuter(),
	}
}

func (s *skopeo) CopyToRegistry(srcImage, dstImage string) error {
	cmd := fmt.Sprintf(templateCopyToRegistry, srcImage, dstImage)
	args := strings.Split(cmd, " ")
	_, err := s.executer.Execute(args[0], args[1:]...)
	return err
}

func (s *skopeo) CopyToFile(imageUrl, imageName, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	cmd := fmt.Sprintf(templateCopyToFile, imageUrl, filePath, imageName)
	args := strings.Split(cmd, " ")
	_, err := s.executer.Execute(args[0], args[1:]...)
	return err
}
