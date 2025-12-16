package skopeo

import (
	"errors"
	"fmt"
	"testing"

	"github.com/openshift/appliance/pkg/consts"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	"github.com/openshift/appliance/pkg/executer"
)

var _ = Describe("Test Skopeo", func() {
	var (
		ctrl         *gomock.Controller
		mockExecuter *executer.MockExecuter
		testSkopeo   Skopeo
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockExecuter = executer.NewMockExecuter(ctrl)
		testSkopeo = NewSkopeo(mockExecuter)
	})

	It("skopeo CopyToFile - success", func() {

		fakePath := "path/to/registry"
		cmd := fmt.Sprintf(templateCopyToFile, consts.RegistryImage, fakePath)
		mockExecuter.EXPECT().Execute(cmd).Return("", nil).Times(1)

		err := testSkopeo.CopyToFile(consts.RegistryImage, consts.RegistryImage, fakePath)
		Expect(err).ToNot(HaveOccurred())
	})

	It("skopeo CopyToFile - failure", func() {
		fakePath := "path/to/registry"
		mockExecuter.EXPECT().Execute(gomock.Any()).Return("", errors.New("some error")).Times(1)

		err := testSkopeo.CopyToFile(consts.RegistryImage, consts.RegistryImage, fakePath)
		Expect(err).To(HaveOccurred())
	})
})

func TestSkopeo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "skopeo_test")
}
