package embeddedconfig

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEmbeddedConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Embedded Config Suite")
}
