package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestISOConfigEmbedder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ISO Config Embedder Suite")
}
