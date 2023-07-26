package genisoimage

import (
	"fmt"

	"github.com/openshift/appliance/pkg/executer"
)

const (
	genDataImageCmd = "genisoimage -J -joliet-long -D -V agentdata -o %s/%s %s"
)

type GenIsoImage interface {
	GenerateImage(imagePath, imageName, dirPath string) error
}

type genisoimage struct {
	executer executer.Executer
}

func NewGenIsoImage(exec executer.Executer) GenIsoImage {
	if exec == nil {
		exec = executer.NewExecuter()
	}

	return &genisoimage{
		executer: exec,
	}
}

func (s *genisoimage) GenerateImage(imagePath, imageName, dirPath string) error {
	_, err := s.executer.Execute(fmt.Sprintf(genDataImageCmd, imagePath, imageName, dirPath))
	return err
}
