package ignition

import (
	"path/filepath"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	"github.com/openshift/installer/pkg/asset/ignition"
)

var _ = Describe("validateSystemManifests", func() {
	configWith := func(path, content string) *igntypes.Config {
		return &igntypes.Config{Storage: igntypes.Storage{Files: []igntypes.File{
			ignition.FileFromString(path, "root", 0644, content),
		}}}
	}

	DescribeTable("checks each manifest in the directory",
		func(content, expectedErr string) {
			path := filepath.Join(systemManifestsPath, "m.yaml")
			err := validateSystemManifests(configWith(path, content), systemManifestsPath)
			if expectedErr == "" {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedErr))
			Expect(err.Error()).To(ContainSubstring(path))
		},
		Entry("valid manifest", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n", ""),
		Entry("leading document separator", "---\napiVersion: v1\nkind: ConfigMap\n", ""),
		Entry("multiple documents", "apiVersion: v1\nkind: A\n---\napiVersion: v1\nkind: B\n", "single YAML document"),
		Entry("empty file", "", "single YAML document"),
		Entry("missing kind", "apiVersion: v1\n", "kind"),
		Entry("missing apiVersion", "kind: ConfigMap\n", "apiVersion"),
		Entry("invalid YAML", "kind: [\n", "invalid"),
	)

	It("ignores files outside the directory", func() {
		cfg := configWith("/etc/assisted/extra-manifests/x.yaml", "kind: [\n")
		cfg.Storage.Files = append(cfg.Storage.Files,
			ignition.FileFromString(filepath.Join(systemManifestsPath, "sub", "x.yaml"), "root", 0644, "kind: [\n"))
		Expect(validateSystemManifests(cfg, systemManifestsPath)).To(Succeed())
	})
})
