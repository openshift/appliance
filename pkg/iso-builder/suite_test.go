// Ginkgo suite entry point — keeps compatibility with legacy ginkgo-based test infrastructure.
package isobuilder

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestISOBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ISO Builder Suite")
}
