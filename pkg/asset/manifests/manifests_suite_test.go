package manifests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func TestManifests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "manifests_test")
}
