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

		_, err = testRelease.GetImageFromRelease(imageName)
		Expect(err).NotTo(HaveOccurred())
	})

	It("GetImageFromRelease - fail oc adm release info", func() {
		imageName := "machine-os-images"
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)

		_, err := testRelease.GetImageFromRelease(imageName)
		Expect(err).To(HaveOccurred())
	})

	Context("GetArchitecture", func() {
		It("should convert amd64 to x86_64 with digest URL", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release@sha256:809c037c016c7c0cbc83ce459ed344a55d65fa6cc0d3aa4d51e9a2d9d0cf7ffa")
			cmd := fmt.Sprintf(templateGetArchitecture, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			mockExecuter.EXPECT().Execute(cmd).Return("amd64", nil).Times(1)

			arch, err := testRelease.GetArchitecture()
			Expect(err).NotTo(HaveOccurred())
			Expect(arch).To(Equal("x86_64"))
		})

		It("should convert amd64 to x86_64 with tag URL", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.21.12-x86_64")
			cmd := fmt.Sprintf(templateGetArchitecture, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			mockExecuter.EXPECT().Execute(cmd).Return("amd64", nil).Times(1)

			arch, err := testRelease.GetArchitecture()
			Expect(err).NotTo(HaveOccurred())
			Expect(arch).To(Equal("x86_64"))
		})

		It("should handle error when getting architecture", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("invalid-url")
			mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("failed to get architecture")).Times(1)

			_, err := testRelease.GetArchitecture()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("IsStableRelease", func() {
		It("should return true for stable release version", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64")
			cmd := fmt.Sprintf(templateGetVersion, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			mockExecuter.EXPECT().Execute(cmd).Return("4.22.0", nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeTrue())
		})

		It("should return true for EC release 4.22.0-ec.5", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-ec.5-x86_64")
			cmd := fmt.Sprintf(templateGetVersion, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			mockExecuter.EXPECT().Execute(cmd).Return("4.22.0-ec.5", nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeTrue())
		})

		It("should return true for RC release version", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-rc.0-x86_64")
			cmd := fmt.Sprintf(templateGetVersion, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			mockExecuter.EXPECT().Execute(cmd).Return("4.22.0-rc.0", nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeTrue())
		})

		It("should return false for nightly release version", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("registry.ci.openshift.org/ocp/release:5.0.0-0.nightly-2026-04-23-082815")
			cmd := fmt.Sprintf(templateGetVersion, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			mockExecuter.EXPECT().Execute(cmd).Return("5.0.0-0.nightly-2026-04-23-082815", nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeFalse())
		})

		It("should return false for CI release 5.0.0-0.ci", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("registry.ci.openshift.org/ocp/release:5.0.0-0.ci-2026-04-23-153053")
			cmd := fmt.Sprintf(templateGetVersion, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			mockExecuter.EXPECT().Execute(cmd).Return("5.0.0-0.ci-2026-04-23-153053", nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeFalse())
		})

		It("should handle error when getting version", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("invalid-url")
			mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("failed to get version")).Times(1)

			_, err := testRelease.IsStableRelease()
			Expect(err).To(HaveOccurred())
		})
	})
})

func TestRelease(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "release_test")
}
