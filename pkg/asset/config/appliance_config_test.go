package config

import (
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("appendDigest", func() {
	const digest = "sha256:abc123"

	It("appends digest to image with no tag", func() {
		Expect(appendDigest("registry.example.com/img", digest)).
			To(Equal("registry.example.com/img@sha256:abc123"))
	})

	It("strips tag before appending digest", func() {
		Expect(appendDigest("registry.example.com/img:tag", digest)).
			To(Equal("registry.example.com/img@sha256:abc123"))
	})

	It("handles registry with port and no tag", func() {
		Expect(appendDigest("registry.example.com:5000/img", digest)).
			To(Equal("registry.example.com:5000/img@sha256:abc123"))
	})

	It("strips tag from image with registry port", func() {
		Expect(appendDigest("registry.example.com:5000/img:tag", digest)).
			To(Equal("registry.example.com:5000/img@sha256:abc123"))
	})
})

var _ = Describe("SanitizeOutput", func() {
	const image = "registry.example.com/img@sha256:abc123"
	const warning = "warning: multiple versions reported for the kubectl: 1.36.2 (must-gather), 1.35.2 (network-tools), 1.36.5 (deployer, tools, cli, cli-artifacts)"

	It("returns output unchanged when there are no warnings", func() {
		Expect(SanitizeOutput(image)).To(Equal(image))
	})

	It("trims surrounding whitespace", func() {
		Expect(SanitizeOutput(image + "\n")).To(Equal(image))
	})

	It("removes a warning following the value", func() {
		Expect(SanitizeOutput(image + "\n" + warning)).To(Equal(image))
	})

	It("removes multiple warnings", func() {
		Expect(SanitizeOutput(warning + "\n" + warning + "\n" + image)).
			To(Equal(image))
	})
})
