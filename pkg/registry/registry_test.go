package registry

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/executer"
)

type ClientMock struct{}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), "127.0.0.1") {
		return &http.Response{StatusCode: 200}, nil
	}
	return nil, errors.New("test client error, unexpected URL")
}

var _ = Describe("Test isVersionAtLeast", func() {
	DescribeTable("version comparison",
		func(versionStr, minVersionStr string, expected bool) {
			result := isVersionAtLeast(versionStr, minVersionStr)
			Expect(result).To(Equal(expected))
		},
		Entry("CI version 4.21.0-0.ci-2025-11-17-124207 >= 4.21", "4.21.0-0.ci-2025-11-17-124207", "4.21", true),
		Entry("CI version 4.21.0-0.ci >= 4.21", "4.21.0-0.ci", "4.21", true),
		Entry("Release version 4.21.0 >= 4.21", "4.21.0", "4.21", true),
		Entry("Release version 4.21.5 >= 4.21", "4.21.5", "4.21", true),
		Entry("Release version 4.22.0 >= 4.21", "4.22.0", "4.21", true),
		Entry("CI version 4.22.0-0.nightly-2025-12-01 >= 4.21", "4.22.0-0.nightly-2025-12-01", "4.21", true),
		Entry("Release version 4.20.0 < 4.21", "4.20.0", "4.21", false),
		Entry("Release version 4.20.10 < 4.21", "4.20.10", "4.21", false),
		Entry("CI version 4.20.0-0.ci-2025-11-17-124207 < 4.21", "4.20.0-0.ci-2025-11-17-124207", "4.21", false),
		Entry("Release version 4.19.0 < 4.21", "4.19.0", "4.21", false),
		Entry("Invalid version string", "invalid", "4.21", false),
		Entry("Invalid minimum version", "4.21.0", "invalid", false),
		Entry("Both invalid", "invalid", "also-invalid", false),
	)
})

var _ = Describe("Test Image Registry", func() {
	var (
		ctrl         *gomock.Controller
		mockExecuter *executer.MockExecuter
		port         = 2345
		uri          = "example.io/foobar/registry:1234"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockExecuter = executer.NewMockExecuter(ctrl)
	})

	It("Start Registry - Valid Config", func() {
		dataDirPath, err := GetRegistryDataPath("/fake/path", "/data")
		Expect(err).NotTo(HaveOccurred())

		mockExecuter.EXPECT().Execute(fmt.Sprintf(registryStartCmd, dataDirPath, port, uri)).Return("", nil).Times(1)
		mockExecuter.EXPECT().Execute(registryStopCmd).Return("", nil).Times(1)

		imageRegistry := NewRegistry(
			RegistryConfig{
				URI:         uri,
				Port:        port,
				Executer:    mockExecuter,
				HTTPClient:  &ClientMock{},
				DataDirPath: dataDirPath,
			})

		err = imageRegistry.StartRegistry()
		Expect(err).ToNot(HaveOccurred())
	})

	It("Start Registry - fail to start", func() {
		dataDirPath, err := GetRegistryDataPath("/fake/path", "/data")
		Expect(err).NotTo(HaveOccurred())

		startCmd := fmt.Sprintf(registryStartCmd, dataDirPath, port, uri)

		mockExecuter.EXPECT().Execute(registryStopCmd).Return("", nil).Times(1)
		mockExecuter.EXPECT().Execute(startCmd).Return("", errors.New("some error")).Times(1)

		imageRegistry := NewRegistry(
			RegistryConfig{
				URI:         uri,
				Port:        port,
				Executer:    mockExecuter,
				HTTPClient:  &ClientMock{},
				DataDirPath: dataDirPath,
			})

		err = imageRegistry.StartRegistry()
		Expect(err).To(HaveOccurred())
	})

	It("Stop Registry - Success", func() {
		mockExecuter.EXPECT().Execute(registryStopCmd).Return("", nil).Times(1)

		imageRegistry := NewRegistry(
			RegistryConfig{
				URI:        uri,
				Port:       port,
				Executer:   mockExecuter,
				HTTPClient: &ClientMock{},
			})

		err := imageRegistry.StopRegistry()
		Expect(err).NotTo(HaveOccurred())
	})

	It("Stop Registry - Fail", func() {
		mockExecuter.EXPECT().Execute(registryStopCmd).Return("", errors.New("some error")).Times(1)

		imageRegistry := NewRegistry(
			RegistryConfig{
				URI:        uri,
				Port:       port,
				Executer:   mockExecuter,
				HTTPClient: &ClientMock{},
			})

		err := imageRegistry.StopRegistry()
		Expect(err).To(HaveOccurred())
	})
})

func TestRegistry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "registry_test")
}
