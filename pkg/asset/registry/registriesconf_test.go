package registry

import (
	"context"
	"testing"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/installer/pkg/asset"
)

var _ = Describe("Test RegistriesConf", func() {
	var (
		fakeFileSystem fstest.MapFS
		deps           asset.Parents
		r              RegistriesConf
	)

	BeforeEach(func() {
		fakeFileSystem = fstest.MapFS{}
		deps = asset.Parents{}

		deps.Add(&config.EnvConfig{}, &config.ApplianceConfig{})
		r = RegistriesConf{
			fSys: fakeFileSystem,
		}
	})

	It("OpenShift CI like mirror file", func() {
		fakeFileSystem[idmsFileName] = createOpenShiftCIMirrorFile()

		err := r.Generate(context.Background(), deps)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(r.File.Data)).To(Equal("unqualified-search-registries = []\n\n[[registry]]\n  location = \"quay-proxy.ci.openshift.org/openshift/ci\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/openshift/release\"\n\n[[registry]]\n  location = \"registry.build05.ci.openshift.org/ci-op-f7f21dkx/stable\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/openshift/release\"\n\n[[registry]]\n  location = \"registry.build05.ci.openshift.org/ci-op-f7f21dkx/release\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/openshift/release-images\"\n"))
	})

	It("Single Yaml", func() {
		fakeFileSystem[idmsFileName] = createSingleYamlIDMSMirrorFile()

		err := r.Generate(context.Background(), deps)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(r.File.Data)).To(Equal("unqualified-search-registries = []\n\n[[registry]]\n  location = \"registry.ci.openshift.org/ocp/release\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/openshift/release-images\"\n\n[[registry]]\n  location = \"quay.io/openshift-release-dev/ocp-v4.0-art-dev\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/openshift/release\"\n"))
	})

	It("Multiple Yaml", func() {
		fakeFileSystem[idmsFileName] = createMultipleYamlIDMSMirrorFile()

		err := r.Generate(context.Background(), deps)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(r.File.Data)).To(Equal("unqualified-search-registries = []\n\n[[registry]]\n  location = \"registry.redhat.io/container-native-virtualization\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/container-native-virtualization\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/container-native-virtualization\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/container-native-virtualization\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/container-native-virtualization\"\n\n[[registry]]\n  location = \"registry.redhat.io/openshift4\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/openshift4\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift4\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/openshift4\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/openshift4\"\n\n[[registry]]\n  location = \"registry.redhat.io/workload-availability\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/workload-availability\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/workload-availability\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/workload-availability\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/workload-availability\"\n\n[[registry]]\n  location = \"registry.redhat.io/migration-toolkit-virtualization\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/migration-toolkit-virtualization\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/migration-toolkit-virtualization\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/migration-toolkit-virtualization\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/migration-toolkit-virtualization\"\n\n[[registry]]\n  location = \"registry.redhat.io/kube-descheduler-operator\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/kube-descheduler-operator\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/kube-descheduler-operator\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/kube-descheduler-operator\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/kube-descheduler-operator\"\n\n[[registry]]\n  location = \"registry.ci.openshift.org/ocp/release\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/openshift/release-images\"\n\n[[registry]]\n  location = \"quay.io/openshift-release-dev/ocp-v4.0-art-dev\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:22625/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"api-int.registry.appliance.openshift.com:22625/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"localhost:22625/openshift/release\"\n"))
	})
})

func TestRegistriesConf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "registriesconf_test")
}

// In an OpenShift CI job, the quay-proxy may be also
// present as an additional mirroring location
func createOpenShiftCIMirrorFile() *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(`apiVersion: config.openshift.io/v1
kind: ImageDigestMirrorSet
metadata:
  name: idms-release-0
spec:
  imageDigestMirrors:
  - mirrors:
    - 127.0.0.1:5005/openshift/release
    source: quay-proxy.ci.openshift.org/openshift/ci
  - mirrors:
    - 127.0.0.1:5005/openshift/release
    source: registry.build05.ci.openshift.org/ci-op-f7f21dkx/stable
  - mirrors:
    - 127.0.0.1:5005/openshift/release-images
    source: registry.build05.ci.openshift.org/ci-op-f7f21dkx/release
status: {}`)}
}

func createSingleYamlIDMSMirrorFile() *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(
			`apiVersion: config.openshift.io/v1
kind: ImageDigestMirrorSet
metadata:
  name: idms-release-0
spec:
  imageDigestMirrors:
  - mirrors:
    - 127.0.0.1:5005/openshift/release-images
    source: registry.ci.openshift.org/ocp/release
  - mirrors:
    - 127.0.0.1:5005/openshift/release
    source: quay.io/openshift-release-dev/ocp-v4.0-art-dev
status: {}`)}
}

func createMultipleYamlIDMSMirrorFile() *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(
			`kind: ImageDigestMirrorSet
metadata:
  name: idms-operator-0
spec:
  imageDigestMirrors:
  - mirrors:
    - 127.0.0.1:5005/container-native-virtualization
    source: registry.redhat.io/container-native-virtualization
  - mirrors:
    - 127.0.0.1:5005/openshift4
    source: registry.redhat.io/openshift4
  - mirrors:
    - 127.0.0.1:5005/workload-availability
    source: registry.redhat.io/workload-availability
  - mirrors:
    - 127.0.0.1:5005/migration-toolkit-virtualization
    source: registry.redhat.io/migration-toolkit-virtualization
  - mirrors:
    - 127.0.0.1:5005/kube-descheduler-operator
    source: registry.redhat.io/kube-descheduler-operator
status: {}
---
apiVersion: config.openshift.io/v1
kind: ImageDigestMirrorSet
metadata:
  name: idms-release-0
spec:
  imageDigestMirrors:
  - mirrors:
    - 127.0.0.1:5005/openshift/release-images
    source: registry.ci.openshift.org/ocp/release
  - mirrors:
    - 127.0.0.1:5005/openshift/release
    source: quay.io/openshift-release-dev/ocp-v4.0-art-dev
status: {}`)}
}
