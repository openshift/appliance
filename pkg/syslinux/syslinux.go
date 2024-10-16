package syslinux

import (
	"fmt"

	"github.com/openshift/appliance/pkg/executer"
)

const (
	isoHybridCmd = "isohybrid -u %s"
)

type IsoHybrid interface {
	Convert(imagePath string) error
}

type isohybrid struct {
	executer executer.Executer
}

func NewIsoHybrid(exec executer.Executer) IsoHybrid {
	if exec == nil {
		exec = executer.NewExecuter()
	}

	return &isohybrid{
		executer: exec,
	}
}

func (s *isohybrid) Convert(imagePath string) error {
	_, err := s.executer.Execute(fmt.Sprintf(isoHybridCmd, imagePath))
	return err
}
