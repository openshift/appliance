package genisoimage

import (
	"fmt"
	"strings"

	"github.com/danielerez/openshift-appliance/pkg/executer"
)

const (
	genDataImageCmd = "genisoimage -J -joliet-long -D -V agentdata -o %s/data.iso %s"
)

type GenIsoImage interface {
	GenerateDataImage(imagePath string, dataPath string) error
}

type genisoimage struct {
	executer executer.Executer
}

func NewGenIsoImage() GenIsoImage {
	return &genisoimage{
		executer: executer.NewExecuter(),
	}
}

func (s *genisoimage) GenerateDataImage(imagePath string, dataPath string) error {
	cmd := fmt.Sprintf(genDataImageCmd, imagePath, dataPath)
	args := strings.Split(cmd, " ")
	_, err := s.executer.Execute(args[0], args[1:]...)
	return err
}
