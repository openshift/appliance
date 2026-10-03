//go:build integration

package integration

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/sirupsen/logrus"

	appliancedata "github.com/openshift/appliance/data"
	isobuilder "github.com/openshift/appliance/pkg/iso-builder"
	installerdata "github.com/openshift/installer/data"
)

func isoBuilderMain() {
	installerdata.Assets = http.FS(appliancedata.EmbeddedAssets)
	if err := isobuilder.RunCommand(); err != nil {
		logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
	}
}

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"openshift-iso-builder":         isoBuilderMain,
		"openshift-iso-builder-patched": isoBuilderMain,
		"iso-config-embedder": func() {
			if err := isobuilder.RunEmbedder(); err != nil {
				logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
			}
		},
	})
}

func setupISOBuilder(env *testscript.Env) error {
	binPath, err := exec.LookPath("openshift-iso-builder")
	if err != nil {
		return err
	}
	env.Setenv("ISO_BUILDER_BIN_RAW", binPath)
	return nil
}

func TestBuild(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:   "testdata/build",
		Setup: setupISOBuilder,
	})
}

func TestShowConfig(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:   "testdata/show-config",
		Setup: setupISOBuilder,
	})
}
