//go:build integration

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/sirupsen/logrus"

	appliancedata "github.com/openshift/appliance/data"
	isobuilder "github.com/openshift/appliance/pkg/iso-builder"
	installerdata "github.com/openshift/installer/data"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"openshift-iso-builder": func() {
			installerdata.Assets = http.FS(appliancedata.EmbeddedAssets)
			if err := isobuilder.RunCommand(); err != nil {
				logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
			}
		},
	})
}

func TestBuild(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/build",
	})
}

func TestShowConfig(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/show-config",
	})
}
