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
		file            *os.File
		tempDir         string
		err             error
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockExecuter = executer.NewMockExecuter(ctrl)
		tempDir, err = filepath.Abs("")
		Expect(err).ToNot(HaveOccurred())
		file, err = os.CreateTemp(tempDir, "registry-config")
		Expect(err).ToNot(HaveOccurred())

		channel := graph.ReleaseChannelStable
		applianceConfig = &config.ApplianceConfig{
			Config: &types.ApplianceConfig{
				PullSecret: "'{\"auths\":{\"\":{\"auth\":\"dXNlcjpwYXNz\"}}}'",
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

	It("MirrorReleaseImages - success", func() {
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", nil).Times(1)

		err = testRelease.MirrorReleaseImages()
		Expect(err).ToNot(HaveOccurred())
	})

	It("MirrorReleaseImages - fail oc mirror", func() {
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)

		err = testRelease.MirrorReleaseImages()
		Expect(err).To(HaveOccurred())
	})

	It("MirrorBootstrapImages - success", func() {
		mockExecuter.EXPECT().Execute(gomock.Any()).Return(fakeCincinnatiMetadata, nil).Times(1)
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", nil).Times(1)
		mockExecuter.EXPECT().TempFile(gomock.Any(), "registry-config").Return(file, nil).Times(1)

		err = testRelease.MirrorBootstrapImages()
		Expect(err).ToNot(HaveOccurred())
	})

	It("MirrorBootstrapImages - fail to generate blocked images list", func() {
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)
		mockExecuter.EXPECT().TempFile(gomock.Any(), "registry-config").Return(file, nil).Times(1)

		err = testRelease.MirrorBootstrapImages()
		Expect(err).To(HaveOccurred())
	})

	It("MirrorBootstrapImages - fail oc mirror", func() {
		mockExecuter.EXPECT().Execute(gomock.Any()).Return(fakeCincinnatiMetadata, nil).Times(1)
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)
		mockExecuter.EXPECT().TempFile(gomock.Any(), "registry-config").Return(file, nil).Times(1)

		err = testRelease.MirrorBootstrapImages()
		Expect(err).To(HaveOccurred())
	})

	It("GetImageFromRelease - success", func() {
		imageName := "machine-os-images"
		cmd := fmt.Sprintf(templateGetImage, imageName, true, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
		cmdWithRegConf := fmt.Sprintf("%s --registry-config=%s", cmd, file.Name())

		mockExecuter.EXPECT().Execute(cmdWithRegConf).Return("", nil).Times(1)
		mockExecuter.EXPECT().TempFile(gomock.Any(), "registry-config").Return(file, nil).Times(1)

		_, err := testRelease.GetImageFromRelease(imageName)
		Expect(err).NotTo(HaveOccurred())
	})

	It("GetImageFromRelease - fail oc adm release info", func() {
		imageName := "machine-os-images"
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)
		mockExecuter.EXPECT().TempFile(gomock.Any(), "registry-config").Return(file, nil).Times(1)

		_, err := testRelease.GetImageFromRelease(imageName)
		Expect(err).To(HaveOccurred())
	})

	It("GetImageFromRelease - fail to create a pull secret file", func() {
		imageName := "machine-os-images"
		mockExecuter.EXPECT().TempFile(gomock.Any(), "registry-config").Return(nil, errors.New("some error")).Times(1)

		_, err := testRelease.GetImageFromRelease(imageName)
		Expect(err).To(HaveOccurred())
	})
})

func TestRelease(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "release_test")
}
