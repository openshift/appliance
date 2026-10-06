//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/sirupsen/logrus"

	appliancedata "github.com/openshift/appliance/data"
	isobuilder "github.com/openshift/appliance/pkg/iso-builder"
	isoconfigembedder "github.com/openshift/appliance/pkg/iso-config-embedder"
	installerdata "github.com/openshift/installer/data"
	"go.podman.io/storage/pkg/reexec"
)

func isoBuilderMain() {
	installerdata.Assets = http.FS(appliancedata.EmbeddedAssets)
	if err := isobuilder.Run(); err != nil {
		logrus.Fatalf("Error executing %s: %v", filepath.Base(os.Args[0]), err)
	}
}

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	testscript.Main(m, map[string]func(){
		"openshift-iso-builder":         isoBuilderMain,
		"openshift-iso-builder-patched": isoBuilderMain,
		"iso-config-embedder": func() {
			if err := isoconfigembedder.Run(); err != nil {
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

	// Override the default /no-home with a real writable HOME inside the test workdir.
	homeDir := filepath.Join(env.WorkDir, "home")
	if err := os.Mkdir(homeDir, 0777); err != nil {
		return err
	}
	for i, v := range env.Vars {
		if v == "HOME=/no-home" {
			env.Vars[i] = fmt.Sprintf("HOME=%s", homeDir)
			break
		}
	}

	// Forward host XDG_RUNTIME_DIR so container runtimes can find their socket.
	if xdgDir := os.Getenv("XDG_RUNTIME_DIR"); xdgDir != "" {
		env.Setenv("XDG_RUNTIME_DIR", xdgDir)
	}

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
