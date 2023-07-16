package genisoimage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/executer"
)

var _ = Describe("Test GenIsoImage", func() {
	var (
		ctrl          *gomock.Controller
		mockExecuter  *executer.MockExecuter
		fakeCachePath = "/path/to/cache"
		fakeDataPath  = "/path/to/data"
		fakeImageName = "testdata.iso"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockExecuter = executer.NewMockExecuter(ctrl)
	})

	It("genisoimage GenerateImage - success", func() {

		fakeCachePath := "/path/to/cache"
		fakeDataPath := "/path/to/data"
		fakeImageName := "testdata.iso"

		copyCmd, copyCmdrgs := executer.FormatCommand(fmt.Sprintf(genDataImageCmd, fakeCachePath, fakeImageName, fakeDataPath))
		mockExecuter.EXPECT().Execute(copyCmd, copyCmdrgs).Return("", nil).Times(1)

		skopeo := NewGenIsoImage(mockExecuter)
		err := skopeo.GenerateImage(fakeCachePath, fakeImageName, fakeDataPath)
		Expect(err).ToNot(HaveOccurred())
	})

	It("genisoimage GenerateImage - failure", func() {
		mockExecuter.EXPECT().Execute(gomock.Any(), gomock.Any()).Return("", errors.New("some error")).Times(1)

		skopeo := NewGenIsoImage(mockExecuter)
		err := skopeo.GenerateImage(fakeCachePath, fakeImageName, fakeDataPath)
		Expect(err).To(HaveOccurred())
	})
})

func TestGenIsoImage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "geoisoimage_test")
}
