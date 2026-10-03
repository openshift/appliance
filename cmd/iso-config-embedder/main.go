package main

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	isobuilder "github.com/openshift/appliance/pkg/iso-builder"
)

func main() {
	if err := isobuilder.RunEmbedder(); err != nil {
		logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
	}
}

