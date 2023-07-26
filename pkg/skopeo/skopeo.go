package skopeo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openshift/appliance/pkg/executer"
)

const (
	templateCopyToFile = "skopeo copy docker://%s docker-archive:%s:%s"
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

	_, err := s.executer.Execute(fmt.Sprintf(templateCopyToFile, imageUrl, filePath, imageName))
	return err
}
