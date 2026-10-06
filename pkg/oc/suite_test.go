package oc

import (
	"testing"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func TestOc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "oc suite")
}
