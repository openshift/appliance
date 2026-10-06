package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	appliancedata "github.com/openshift/appliance/data"

	isobuilder "github.com/openshift/appliance/pkg/iso-builder"
	installerdata "github.com/openshift/installer/data"

	"go.podman.io/storage/pkg/reexec"
)

func main() {
	// Required by go.podman.io/storage: the containers-storage library
	// re-executes the current process as a helper subprocess for overlay
	// operations (e.g. applying tar layers). Without this call, any code
	// path that opens a storage.Store will panic.
	if reexec.Init() {
		return
	}

	// Embed the data/ directory (systemd units, scripts, udev rules) into
	// the binary so iso-builder runs standalone without needing the repo
	// layout on disk.
	installerdata.Assets = http.FS(appliancedata.EmbeddedAssets)

	if err := isobuilder.Run(); err != nil {
		logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
	}
}
