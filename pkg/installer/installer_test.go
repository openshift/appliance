package installer

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/graph"
	"github.com/openshift/appliance/pkg/types"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/executer"
)

var _ = Describe("Test Installer", func() {
	var (
		ctrl          *gomock.Controller
		mockExecuter  *executer.MockExecuter
		testInstaller Installer
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockExecuter = executer.NewMockExecuter(ctrl)
	})

	It("GetInstallerDownloadURL - x86_64 stable", func() {
		version := "4.13.1"
		channel := graph.ReleaseChannelStable
		cpuArc := swag.String(config.CpuArchitectureX86)
		installerConfig := InstallerConfig{
			Executer:  mockExecuter,
			EnvConfig: &config.EnvConfig{},
			ApplianceConfig: &config.ApplianceConfig{
				Config: &types.ApplianceConfig{
					OcpRelease: types.ReleaseImage{
						Version:         version,
						Channel:         &channel,
						CpuArchitecture: cpuArc,
					},
				},
			},
		}
		testInstaller = NewInstaller(installerConfig)

		res, err := testInstaller.GetInstallerDownloadURL()
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(fmt.Sprintf(templateInstallerDownloadURL, "4", swag.StringValue(cpuArc), version)))
	})

	It("GetInstallerDownloadURL - aarch64 candidate", func() {
		version := "4.13.2"
		channel := graph.ReleaseChannelCandidate
		cpuArc := swag.String(config.CpuArchitectureAARCH64)
		installerConfig := InstallerConfig{
			Executer:  mockExecuter,
			EnvConfig: &config.EnvConfig{},
			ApplianceConfig: &config.ApplianceConfig{
				Config: &types.ApplianceConfig{
					OcpRelease: types.ReleaseImage{
						Version:         version,
						Channel:         &channel,
						CpuArchitecture: cpuArc,
					},
				},
			},
		}
		testInstaller = NewInstaller(installerConfig)

		res, err := testInstaller.GetInstallerDownloadURL()
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(fmt.Sprintf(templateInstallerDownloadURL, "4", swag.StringValue(cpuArc), version)))
	})

	It("CreateUnconfiguredIgnition - DebugBaseIgnition: false", func() {
		version := "4.13.1"
		channel := graph.ReleaseChannelStable
		cpuArc := swag.String(config.CpuArchitectureX86)

		tmpDir, err := filepath.Abs("")
		Expect(err).ToNot(HaveOccurred())
		cmd := fmt.Sprintf(templateUnconfiguredIgnitionBinary, installerBinaryName, tmpDir)
		mockExecuter.EXPECT().Execute(cmd).Return("", nil).Times(1)

		installerConfig := InstallerConfig{
			Executer: mockExecuter,
			EnvConfig: &config.EnvConfig{
				DebugBaseIgnition: false,
				TempDir:           tmpDir,
			},
			ApplianceConfig: &config.ApplianceConfig{
				Config: &types.ApplianceConfig{
					OcpRelease: types.ReleaseImage{
						Version:         version,
						Channel:         &channel,
						CpuArchitecture: cpuArc,
					},
				},
			},
		}
		testInstaller = NewInstaller(installerConfig)

		res, err := testInstaller.CreateUnconfiguredIgnition()
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(fmt.Sprintf("%s/unconfigured-agent.ign", tmpDir)))
	})

	It("CreateUnconfiguredIgnition - DebugBaseIgnition: true", func() {
		version := "4.13.1"
		channel := graph.ReleaseChannelStable
		cpuArc := swag.String(config.CpuArchitectureX86)
		tmpDir := "/path/to/tempdir"

		cmd := fmt.Sprintf(templateUnconfiguredIgnitionBinary, installerBinaryName, tmpDir)
		mockExecuter.EXPECT().Execute(cmd).Return("", nil).Times(1)

		installerConfig := InstallerConfig{
			Executer: mockExecuter,
			EnvConfig: &config.EnvConfig{
				DebugBaseIgnition: true,
				TempDir:           tmpDir,
			},
			ApplianceConfig: &config.ApplianceConfig{
				Config: &types.ApplianceConfig{
					OcpRelease: types.ReleaseImage{
						Version:         version,
						Channel:         &channel,
						CpuArchitecture: cpuArc,
					},
				},
			},
		}
		testInstaller = NewInstaller(installerConfig)

		res, err := testInstaller.CreateUnconfiguredIgnition()
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(filepath.Join(tmpDir, unconfiguredIgnitionFileName)))
	})

})

func TestInstaller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "installer_test")
}
