package registry

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/consts"
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
		Entry("CI version 4.21.0-0.ec.3 >= 4.21", "4.21.0-0.ec.3", "4.21", true),
		Entry("Release version 4.21.0 >= 4.21", "4.21.0", "4.21", true),
		Entry("Release version 4.21.5 >= 4.21", "4.21.5", "4.21", true),
		Entry("Release version 4.22.0 >= 4.21", "4.22.0", "4.21", true),
		Entry("CI version 4.22.0-0.nightly-2025-12-01 >= 4.21", "4.22.0-0.nightly-2025-12-01", "4.21", true),
		Entry("Release version 4.20.10 < 4.21", "4.20.10", "4.21", false),
		Entry("CI version 4.20.0-0.ci-2025-11-17-124207 < 4.21", "4.20.0-0.ci-2025-11-17-124207", "4.21", false),
		Entry("Invalid version string", "invalid", "4.21", false),
		Entry("Invalid minimum version", "4.21.0", "invalid", false),
		Entry("Both invalid", "invalid", "also-invalid", false),
	)
})

var _ = Describe("RegistryCacheDigestKey", func() {
	DescribeTable("maps pull spec to cache subdirectory name",
		func(source, want string) {
			Expect(RegistryCacheDigestKey(source)).To(Equal(want))
		},
		Entry("repo@sha256:<hex> uses hash portion only",
			"quay.io/ocp/release@sha256:0f57ec0abf6762a265f2e4f5523c170735c3a61be89032b71709ccf9daf430ce",
			"0f57ec0abf6762a265f2e4f5523c170735c3a61be89032b71709ccf9daf430ce"),
		Entry("uses last @ in reference",
			"registry.io/ns/w@x@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Entry("digest with sha512 algorithm prefix",
			"example.io/img@sha512:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		Entry("digest part without algo: returns whole digest segment",
			"registry.io/img@opaque-digest-value",
			"opaque-digest-value"),
		Entry("built-in localhost registry (no @)",
			consts.RegistryImage,
			"internal"),
		Entry("tag-only reference (no @)",
			"quay.io/foo/bar:latest",
			"internal"),
	)
})

var _ = Describe("isRegistryImageCacheHit", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "registry-cache-hit-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })
	})

	It("returns false for a missing path", func() {
		Expect(isRegistryImageCacheHit(filepath.Join(tmpDir, "nonexistent"))).To(BeFalse())
	})

	It("returns false for an empty directory", func() {
		empty := filepath.Join(tmpDir, "empty")
		Expect(os.MkdirAll(empty, 0o755)).To(Succeed())
		Expect(isRegistryImageCacheHit(empty)).To(BeFalse())
	})

	It("returns false for a regular file", func() {
		f := filepath.Join(tmpDir, "file")
		Expect(os.WriteFile(f, []byte("x"), 0o644)).To(Succeed())
		Expect(isRegistryImageCacheHit(f)).To(BeFalse())
	})

	It("returns true when the directory has at least one entry", func() {
		regDir := filepath.Join(tmpDir, "registry-dir")
		Expect(os.MkdirAll(regDir, 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(regDir, "manifest.json"), []byte("{}"), 0o644)).To(Succeed())
		Expect(isRegistryImageCacheHit(regDir)).To(BeTrue())
	})
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
