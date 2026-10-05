package main

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	isoconfigembedder "github.com/openshift/appliance/pkg/iso-config-embedder"
)

func main() {
	if err := isoconfigembedder.Run(); err != nil {
		logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
	}
}

