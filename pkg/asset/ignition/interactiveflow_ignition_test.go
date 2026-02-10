package ignition

import (
	"testing"

	"github.com/coreos/ignition/v2/config/v3_2/types"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	"github.com/vincent-petithory/dataurl"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Test InteractiveFlow Ignition", func() {
	var (
		ign *types.Config
	)

	BeforeEach(func() {
		ign = &types.Config{
			Storage: types.Storage{},
		}
	})

	It("Default additional files", func() {
		i := *NewInteractiveFlowIgnition("4.20.5-x86_64")
		i.AppendToIgnition(ign)
		Expect(ign.Storage.Files).To(HaveLen(3))

		for _, f := range []string{
			"/etc/assisted/interactive-ui",
			"/etc/assisted/no-config-image",
			"/etc/assisted/extra-manifests/internalreleaseimage.yaml",
		} {
			data, err := ignitionGetFileData(ign, f)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())
		}
	})

	DescribeTable("Release names",
		func(releaseVersion string, expectedReleaseStr string) {
			i := *NewInteractiveFlowIgnition(releaseVersion)
			i.AppendToIgnition(ign)

			data, err := ignitionGetFileData(ign, "/etc/assisted/extra-manifests/internalreleaseimage.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())

			type IRI struct {
				Spec struct {
					Releases []struct {
						Name string `yaml:"name"`
					} `yaml:"releases"`
				} `yaml:"spec"`
			}
			var iri IRI
			err = yaml.Unmarshal(data, &iri)
			Expect(err).NotTo(HaveOccurred())
			Expect(iri.Spec.Releases[0].Name).To(Equal(expectedReleaseStr))
		},
		Entry("valid release name", "4.21.0-ec.3-x86_64", "ocp-release-bundle-4.21.0-ec.3-x86_64"),
		Entry("valid release name", "4.20.5-x86_64", "ocp-release-bundle-4.20.5-x86_64"),
		Entry("valid release name", "4.14.0-0.nightly-2025-11-23-025204", "ocp-release-bundle-4.14.0-0.nightly-2025-11-23-025204"),
		Entry("valid release name", "4.21.0-ec.2-s390x", "ocp-release-bundle-4.21.0-ec.2-s390x"),
		Entry("valid release name", "4.15.0-0.ci-2025-11-22-162639", "ocp-release-bundle-4.15.0-0.ci-2025-11-22-162639"),
		Entry("trim releases longer than 64 chars", "4.22.0-0.ci-2026-02-09-204741-test-ci-op-phx0mrh8-latest", "ocp-release-bundle-4.22.0-0.ci-2026-02-09-204741-test-ci-op-phx0"),
	)
})

func ignitionGetFileData(ign *types.Config, filePath string) ([]byte, error) {
	for _, f := range ign.Storage.Files {
		if f.Path == filePath {
			if f.Contents.Source != nil {
				contents, err := dataurl.DecodeString(*f.Contents.Source)
				if err != nil {
					return nil, err
				}
				return contents.Data, err
			}
		}
	}
	return nil, nil
}

func TestIgnition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ignition_test")
}
