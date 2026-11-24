package ignition

import (
	"path/filepath"

	"github.com/go-openapi/swag/conv"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("BootstrapIgnition manifests path", func() {
	DescribeTable("manifestsPathFor",
		func(enableInteractiveFlow *bool, expected string) {
			Expect(manifestsPathFor(enableInteractiveFlow)).To(Equal(expected))
		},
		Entry("interactive flow", conv.Pointer(true), systemManifestsPath),
		Entry("non-interactive flow", conv.Pointer(false), extraManifestsPath),
		Entry("unset", nil, extraManifestsPath),
	)

	DescribeTable("writes generated manifests under manifestsPath",
		func(dir string, write func(i *BootstrapIgnition) error, fileName string) {
			i := &BootstrapIgnition{manifestsPath: dir}
			Expect(write(i)).To(Succeed())
			Expect(hasStorageFile(i.Config, filepath.Join(dir, fileName))).To(BeTrue())
		},
		Entry("OperatorHub in system-manifests", systemManifestsPath,
			func(i *BootstrapIgnition) error { return i.disableDefaultCatalogSources() }, "operatorhub-cluster.yaml"),
		Entry("OperatorHub in extra-manifests", extraManifestsPath,
			func(i *BootstrapIgnition) error { return i.disableDefaultCatalogSources() }, "operatorhub-cluster.yaml"),
		Entry("core password MachineConfig in system-manifests", systemManifestsPath,
			func(i *BootstrapIgnition) error { return i.setCoreUserPass("master", "hash") }, "99-master-set-core-pass.yaml"),
		Entry("core password MachineConfig in extra-manifests", extraManifestsPath,
			func(i *BootstrapIgnition) error { return i.setCoreUserPass("master", "hash") }, "99-master-set-core-pass.yaml"),
	)

	It("keeps post-installation CRs under extra-manifests", func() {
		Expect(filepath.Join(extraManifestsPath, postInstallationCrsDir)).To(Equal("/etc/assisted/extra-manifests/post-installation"))
	})
})
