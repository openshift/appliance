package releasebundle

import (
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func TestReleasebundle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "releasebundle suite")
}
