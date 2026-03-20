package syslinux

import (
	"errors"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/executer"
)

var _ = Describe("Test IsoHybrid", func() {
	var (
		ctrl          *gomock.Controller
		mockExecuter  *executer.MockExecuter
		testIsoHybrid IsoHybrid
		fakeImagePath = "/path/to/testdata.iso"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockExecuter = executer.NewMockExecuter(ctrl)
		testIsoHybrid = NewIsoHybrid(mockExecuter)
	})

	It("isohybrid Convert - success", func() {

		fakeImagePath = "/path/to/testdata.iso"

		cmd := fmt.Sprintf(isoHybridCmd, fakeImagePath)
		mockExecuter.EXPECT().Execute(cmd).Return("", nil).Times(1)

		err := testIsoHybrid.Convert(fakeImagePath)
		Expect(err).ToNot(HaveOccurred())
	})

	It("isohybrid Convert - failure", func() {
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)

		err := testIsoHybrid.Convert(fakeImagePath)
		Expect(err).To(HaveOccurred())
	})
})

func TestGenIsoImage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "isohybrid_test")
}
