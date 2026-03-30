package release

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/swag"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
	"github.com/openshift/appliance/pkg/graph"
	"github.com/openshift/appliance/pkg/types"
)

type FakeOS struct{}

func (FakeOS) MkdirTemp(dir, prefix string) (string, error) {
	return os.MkdirTemp(dir, prefix)
}

func (FakeOS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (FakeOS) Remove(name string) error {
	return os.Remove(name)
}

func (FakeOS) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (FakeOS) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (FakeOS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return nil
}

func (FakeOS) ReadFile(name string) ([]byte, error) {
	return nil, nil
}

func (FakeOS) RemoveAll(path string) error {
	return nil
}

var _ = Describe("Test Release", func() {
	var (
		ctrl            *gomock.Controller
		mockExecuter    *executer.MockExecuter
		applianceConfig *config.ApplianceConfig
		testRelease     Release
		tempDir         string
		err             error
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockExecuter = executer.NewMockExecuter(ctrl)
		tempDir, err = filepath.Abs("")
		Expect(err).ToNot(HaveOccurred())

		channel := graph.ReleaseChannelStable
		applianceConfig = &config.ApplianceConfig{
			Config: &types.ApplianceConfig{
				OcpRelease: types.ReleaseImage{
					CpuArchitecture: swag.String(config.CpuArchitectureX86),
					Version:         "4.13.1",
					Channel:         &channel,
				},
				ImageRegistry: &types.ImageRegistry{
					Port: swag.Int(5123),
				},
			},
		}

		coreOSConfig := ReleaseConfig{
			OSInterface:     &FakeOS{},
			ApplianceConfig: applianceConfig,
			Executer:        mockExecuter,
			EnvConfig: &config.EnvConfig{
				TempDir: tempDir,
			},
		}
		testRelease = NewRelease(coreOSConfig)
	})

	AfterEach(func() {
		// Clean up scripts/mirror directory created by RenderTemplateFile during tests
		scriptsDir := filepath.Join(tempDir, "scripts")
		err := os.RemoveAll(scriptsDir)
		Expect(err).ToNot(HaveOccurred())
	})

	It("MirrorInstallImages - success", func() {
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", nil).Times(1)

		err = testRelease.MirrorInstallImages()
		Expect(err).ToNot(HaveOccurred())
	})

	It("MirrorInstallImages - fail oc mirror", func() {
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)

		err = testRelease.MirrorInstallImages()
		Expect(err).To(HaveOccurred())
	})

	It("GetImageFromRelease - success", func() {
		imageName := "machine-os-images"
		cmd := fmt.Sprintf(templateGetImage, imageName, true, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
		mockExecuter.EXPECT().Execute(cmd).Return("", nil).Times(1)

		_, err := testRelease.GetImageFromRelease(imageName)
		Expect(err).NotTo(HaveOccurred())
	})

	It("GetImageFromRelease - fail oc adm release info", func() {
		imageName := "machine-os-images"
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)

		_, err := testRelease.GetImageFromRelease(imageName)
		Expect(err).To(HaveOccurred())
	})
})

func TestFindMappingFileInMirrorWorkspace(t *testing.T) {
	t.Run("finds nested mapping.txt", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "a", "b")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		want := "x=y\n"
		if err := os.WriteFile(filepath.Join(sub, "mapping.txt"), []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		b, err := FindMappingFileInMirrorWorkspace(dir)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != want {
			t.Fatalf("got %q want %q", b, want)
		}
	})
	t.Run("missing root returns nil", func(t *testing.T) {
		b, err := FindMappingFileInMirrorWorkspace(filepath.Join(t.TempDir(), "nope"))
		if err != nil {
			t.Fatal(err)
		}
		if b != nil {
			t.Fatal("expected nil bytes")
		}
	})
	t.Run("empty tree returns nil", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "x"), 0o755); err != nil {
			t.Fatal(err)
		}
		b, err := FindMappingFileInMirrorWorkspace(dir)
		if err != nil {
			t.Fatal(err)
		}
		if b != nil {
			t.Fatal("expected nil bytes")
		}
	})
	t.Run("finds working-dir/dry-run/mapping.txt", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "working-dir", "dry-run")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		want := "registry/a=b\n"
		if err := os.WriteFile(filepath.Join(sub, "mapping.txt"), []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		b, err := FindMappingFileInMirrorWorkspace(dir)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != want {
			t.Fatalf("got %q want %q", b, want)
		}
	})
}

func TestRelease(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "release_test")
}
