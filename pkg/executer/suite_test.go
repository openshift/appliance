package executer

import (
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func TestExecuter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Executer")
}
