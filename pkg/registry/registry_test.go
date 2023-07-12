package registry

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2/dsl/core"
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

		startCmd, startCmdrgs := executer.FormatCommand(fmt.Sprintf(registryStartCmd, dataDirPath, port, uri))
		stopCmd, stopCmdrgs := executer.FormatCommand(registryStopCmd)

		mockExecuter.EXPECT().Execute(stopCmd, stopCmdrgs).Return("", nil).Times(1)
		mockExecuter.EXPECT().Execute(startCmd, startCmdrgs).Return("", nil).Times(1)

		imageRegistry := NewRegistry(
			RegistryConfig{
				URI:        uri,
				Port:       port,
				Executer:   mockExecuter,
				HTTPClient: &ClientMock{},
			})

		err = imageRegistry.StartRegistry(dataDirPath)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Start Registry - fail to start", func() {
		dataDirPath, err := GetRegistryDataPath("/fake/path", "/data")
		Expect(err).NotTo(HaveOccurred())

		startCmd, startCmdrgs := executer.FormatCommand(fmt.Sprintf(registryStartCmd, dataDirPath, port, uri))
		stopCmd, stopCmdrgs := executer.FormatCommand(registryStopCmd)

		mockExecuter.EXPECT().Execute(stopCmd, stopCmdrgs).Return("", nil).Times(1)
		mockExecuter.EXPECT().Execute(startCmd, startCmdrgs).Return("", errors.New("some error")).Times(1)

		imageRegistry := NewRegistry(
			RegistryConfig{
				URI:        uri,
				Port:       port,
				Executer:   mockExecuter,
				HTTPClient: &ClientMock{},
			})

		err = imageRegistry.StartRegistry(dataDirPath)
		Expect(err).To(HaveOccurred())
	})

	It("Stop Registry - Success", func() {
		stopCmd, stopCmdrgs := executer.FormatCommand(registryStopCmd)

		mockExecuter.EXPECT().Execute(stopCmd, stopCmdrgs).Return("", nil).Times(1)

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
		stopCmd, stopCmdrgs := executer.FormatCommand(registryStopCmd)

		mockExecuter.EXPECT().Execute(stopCmd, stopCmdrgs).Return("", errors.New("some error")).Times(1)

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
