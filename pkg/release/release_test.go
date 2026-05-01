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
		// Mock IsStableRelease call
		metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
		jsonOutput := `{"metadata":{"version":"4.13.1"}}`
		mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

		// Mock oc mirror command
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", nil).Times(1)

		err = testRelease.MirrorInstallImages()
		Expect(err).ToNot(HaveOccurred())
	})

	It("MirrorInstallImages - fail oc mirror", func() {
		// Mock IsStableRelease call
		metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
		jsonOutput := `{"metadata":{"version":"4.13.1"}}`
		mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

		// Mock oc mirror command failure
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)

		err = testRelease.MirrorInstallImages()
		Expect(err).To(HaveOccurred())
	})

	Context("MirrorInstallImages with signature handling", func() {
		It("should add --ignore-release-signature for CI release", func() {
			// Set up CI release
			applianceConfig.Config.OcpRelease.URL = swag.String("registry.ci.openshift.org/ocp/release:5.0.0-0.ci-2026-04-23-153053")

			// Expect IsStableRelease call (returns CI release metadata)
			metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"5.0.0-0.ci-2026-04-23-153053"}}`
			mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

			// Expect oc mirror command with --ignore-release-signature flag
			mockExecuter.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
				Expect(cmd).To(ContainSubstring("--ignore-release-signature"))
				return "", nil
			}).Times(1)

			err = testRelease.MirrorInstallImages()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT add --ignore-release-signature for stable release", func() {
			// Set up stable release
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64")

			// Expect IsStableRelease call (returns stable release metadata)
			metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"4.22.0"}}`
			mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

			// Expect oc mirror command WITHOUT --ignore-release-signature flag
			mockExecuter.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
				Expect(cmd).ToNot(ContainSubstring("--ignore-release-signature"))
				return "", nil
			}).Times(1)

			err = testRelease.MirrorInstallImages()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should add --ignore-release-signature for nightly release", func() {
			// Set up nightly release
			applianceConfig.Config.OcpRelease.URL = swag.String("registry.ci.openshift.org/ocp/release:5.0.0-0.nightly-2026-04-23-082815")

			// Expect IsStableRelease call (returns nightly release metadata)
			metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"5.0.0-0.nightly-2026-04-23-082815"}}`
			mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

			// Expect oc mirror command with --ignore-release-signature flag
			mockExecuter.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
				Expect(cmd).To(ContainSubstring("--ignore-release-signature"))
				return "", nil
			}).Times(1)

			err = testRelease.MirrorInstallImages()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT add --ignore-release-signature for EC release", func() {
			// Set up EC release
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-ec.5-x86_64")

			// Expect IsStableRelease call (returns EC release metadata)
			metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"4.22.0-ec.5"}}`
			mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

			// Expect oc mirror command WITHOUT --ignore-release-signature flag
			mockExecuter.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
				Expect(cmd).ToNot(ContainSubstring("--ignore-release-signature"))
				return "", nil
			}).Times(1)

			err = testRelease.MirrorInstallImages()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT add --ignore-release-signature for RC release", func() {
			// Set up RC release
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-rc.0-x86_64")

			// Expect IsStableRelease call (returns RC release metadata)
			metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"4.22.0-rc.0"}}`
			mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

			// Expect oc mirror command WITHOUT --ignore-release-signature flag
			mockExecuter.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
				Expect(cmd).ToNot(ContainSubstring("--ignore-release-signature"))
				return "", nil
			}).Times(1)

			err = testRelease.MirrorInstallImages()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("GetMappingFile with signature handling", func() {
		It("should add --ignore-release-signature for CI release", func() {
			// Set up CI release
			applianceConfig.Config.OcpRelease.URL = swag.String("registry.ci.openshift.org/ocp/release:5.0.0-0.ci-2026-04-23-153053")

			// Expect IsStableRelease call
			metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"5.0.0-0.ci-2026-04-23-153053"}}`
			mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

			// Expect dry-run command with --ignore-release-signature flag
			mockExecuter.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
				Expect(cmd).To(ContainSubstring("--ignore-release-signature"))
				Expect(cmd).To(ContainSubstring("--dry-run"))
				return "", nil
			}).Times(1)

			_, err = testRelease.GetMappingFile()
			// FakeOS.ReadFile returns nil, nil so this should succeed
			Expect(err).ToNot(HaveOccurred())
		})

		It("should NOT add --ignore-release-signature for stable release", func() {
			// Set up stable release
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64")

			// Expect IsStableRelease call
			metadataCmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"4.22.0"}}`
			mockExecuter.EXPECT().Execute(metadataCmd).Return(jsonOutput, nil).Times(1)

			// Expect dry-run command WITHOUT --ignore-release-signature flag
			mockExecuter.EXPECT().Execute(gomock.Any()).DoAndReturn(func(cmd string) (string, error) {
				Expect(cmd).ToNot(ContainSubstring("--ignore-release-signature"))
				Expect(cmd).To(ContainSubstring("--dry-run"))
				return "", nil
			}).Times(1)

			_, err = testRelease.GetMappingFile()
			// FakeOS.ReadFile returns nil, nil so this should succeed
			Expect(err).ToNot(HaveOccurred())
		})
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
	Context("IsStableRelease", func() {
		It("should return true for stable release version", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64")
			cmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"4.22.0"}}`
			mockExecuter.EXPECT().Execute(cmd).Return(jsonOutput, nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeTrue())
		})

		It("should return true for EC release", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-ec.5-x86_64")
			cmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"4.22.0-ec.5"}}`
			mockExecuter.EXPECT().Execute(cmd).Return(jsonOutput, nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeTrue())
		})

		It("should return true for RC release version", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("quay.io/openshift-release-dev/ocp-release:4.22.0-rc.0-x86_64")
			cmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"4.22.0-rc.0"}}`
			mockExecuter.EXPECT().Execute(cmd).Return(jsonOutput, nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeTrue())
		})

		It("should return false for nightly release version", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("registry.ci.openshift.org/ocp/release:5.0.0-0.nightly-2026-04-23-082815")
			cmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"5.0.0-0.nightly-2026-04-23-082815"}}`
			mockExecuter.EXPECT().Execute(cmd).Return(jsonOutput, nil).Times(1)

			isStable, err := testRelease.IsStableRelease()
			Expect(err).NotTo(HaveOccurred())
			Expect(isStable).To(BeFalse())
		})

		It("should return false for CI release", func() {
			applianceConfig.Config.OcpRelease.URL = swag.String("registry.ci.openshift.org/ocp/release:5.0.0-0.ci-2026-04-23-153053")
			cmd := fmt.Sprintf(templateGetMetadata, swag.StringValue(applianceConfig.Config.OcpRelease.URL))
			jsonOutput := `{"metadata":{"version":"5.0.0-0.ci-2026-04-23-153053"}}`
			mockExecuter.EXPECT().Execute(cmd).Return(jsonOutput, nil).Times(1)

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
