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

	It("Single Yaml", func() {
		fakeFileSystem[idmsFileName] = createSingleYamlIDMSMirrorFile()

		err := r.Generate(context.Background(), deps)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(r.File.Data)).To(Equal("unqualified-search-registries = []\n\n[[registry]]\n  location = \"registry.ci.openshift.org/ocp/release\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5000/openshift/release-images\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release-images\"\n\n[[registry]]\n  location = \"quay.io/openshift-release-dev/ocp-v4.0-art-dev\"\n  prefix = \"\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5000/openshift/release\"\n\n  [[registry.mirror]]\n    location = \"registry.appliance.openshift.com:5001/openshift/release\"\n"))
	})

	It("Multiple Yaml", func() {
		fakeFileSystem[idmsFileName] = createMultipleYamlIDMSMirrorFile()

		err := r.Generate(context.Background(), deps)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(r.File.Data)).Should(ContainSubstring("registry.ci.openshift.org"))
	})

})

func TestRegistriesConf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "registriesconf_test")
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
    - registry.appliance.openshift.com:5000/openshift/release-images
    source: registry.ci.openshift.org/ocp/release
  - mirrors:
    - registry.appliance.openshift.com:5000/openshift/release
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
    - registry.appliance.openshift.com:5000/container-native-virtualization
    source: registry.redhat.io/container-native-virtualization
  - mirrors:
    - registry.appliance.openshift.com:5000/openshift4
    source: registry.redhat.io/openshift4
  - mirrors:
    - registry.appliance.openshift.com:5000/workload-availability
    source: registry.redhat.io/workload-availability
  - mirrors:
    - registry.appliance.openshift.com:5000/migration-toolkit-virtualization
    source: registry.redhat.io/migration-toolkit-virtualization
  - mirrors:
    - registry.appliance.openshift.com:5000/kube-descheduler-operator
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
    - registry.appliance.openshift.com:5000/openshift/release-images
    source: registry.ci.openshift.org/ocp/release
  - mirrors:
    - registry.appliance.openshift.com:5000/openshift/release
    source: quay.io/openshift-release-dev/ocp-v4.0-art-dev
status: {}`)}
}
