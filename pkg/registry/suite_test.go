package registry

import (
	"testing"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestRegistry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry")
}

var logger logr.Logger

var _ = BeforeSuite(func() {
	logrusLogger := logrus.New()
	logrusLogger.Level = logrus.DebugLevel
	logrusLogger.Out = GinkgoWriter
	logrusLogger.Formatter = &logrus.JSONFormatter{
		PrettyPrint: true,
	}
	logger = logrusr.New(logrusLogger)
})
