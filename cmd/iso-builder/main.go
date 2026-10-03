package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	appliancedata "github.com/openshift/appliance/data"

	isobuilder "github.com/openshift/appliance/pkg/iso-builder"
	installerdata "github.com/openshift/installer/data"
)

func main() {
	// Embed the data/ directory (systemd units, scripts, udev rules) into
	// the binary so iso-builder runs standalone without needing the repo
	// layout on disk.
	installerdata.Assets = http.FS(appliancedata.EmbeddedAssets)

	if err := isobuilder.RunCommand(); err != nil {
		logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
	}
}
