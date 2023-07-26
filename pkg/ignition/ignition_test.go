package ignitionutil

import (
	"errors"
	"os"
	"testing"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	assetignition "github.com/openshift/installer/pkg/asset/ignition"
)

const (
	fakeIgnition32 = `{
		"ignition": {
		  "config": {},
		  "version": "3.2.0"
		},
		"storage": {
		  "files": []
		}
	  }`

	fakeIgnition99 = `{
		"ignition": {
		  "config": {},
		  "version": "9.9.0"
		},
		"storage": {
		  "files": []
		}
	  }`
)

type FakeOS struct{}

func (FakeOS) MkdirTemp(dir, prefix string) (string, error) {
	return os.MkdirTemp(dir, prefix)
}

func (FakeOS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (FakeOS) Remove(name string) error {
	return os.Remove(name)
}

func (FakeOS) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (FakeOS) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (FakeOS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (FakeOS) ReadFile(name string) ([]byte, error) {
	if name == "/path/to/ignition32-file" {
		return []byte(fakeIgnition32), nil
	}
	if name == "/path/to/ignition99-file" {
		return []byte(fakeIgnition99), nil
	}
	return nil, errors.New("file not found")
}

func (FakeOS) RemoveAll(path string) error {
	return nil
}

var _ = Describe("Test Ignition", func() {
	var (
		testIgnition Ignition
	)

	BeforeEach(func() {
		coreOSConfig := IgnitionConfig{
			OSInterface: &FakeOS{},
		}
		testIgnition = NewIgnition(coreOSConfig)
	})
	//
	It("ParseIgnitionFile - file not found", func() {
		_, err := testIgnition.ParseIgnitionFile("/bad/path/filename")
		Expect(err).To(HaveOccurred())
	})

	It("ParseIgnitionFile - bad ignition version", func() {
		_, err := testIgnition.ParseIgnitionFile("/path/to/ignition99-file")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("unsupported config version"))
	})

	It("ParseIgnitionFile - ignition v3.2.0 success", func() {
		config32, err := testIgnition.ParseIgnitionFile("/path/to/ignition32-file")
		Expect(err).NotTo(HaveOccurred())
		Expect(config32.Ignition.Version).To(Equal("3.2.0"))
	})

	It("MergeIgnitionConfig - success", func() {
		fakeUser := "core"
		fakePass := swag.String("fakePwdHash")
		fakeDevice := "/boot"
		fakeFormat := swag.String("ext4")
		fakePath := swag.String("/dev/disk/by-partlabel/boot")
		fakeInstallConfig := igntypes.Config{
			Passwd: igntypes.Passwd{
				Users: []igntypes.PasswdUser{
					{
						Name:         fakeUser,
						PasswordHash: fakePass,
					},
				},
			},
			Ignition: igntypes.Ignition{
				Version: igntypes.MaxVersion.String(),
			},
			Storage: igntypes.Storage{
				Files: []igntypes.File{assetignition.FileFromBytes("/path/to/file",
					"root", 0644, []byte("foobar")),
				},
				Filesystems: []igntypes.Filesystem{
					{
						Device: fakeDevice,
						Format: fakeFormat,
						Path:   fakePath,
					},
				},
			},
		}

		config32, err := testIgnition.ParseIgnitionFile("/path/to/ignition32-file")
		Expect(err).NotTo(HaveOccurred())
		mergedConfig, err := testIgnition.MergeIgnitionConfig(config32, &fakeInstallConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergedConfig.Ignition.Version).To(Equal(igntypes.MaxVersion.String()))
		Expect(mergedConfig.Passwd.Users[0].Name).To(Equal(fakeUser))
		Expect(mergedConfig.Passwd.Users[0].PasswordHash).To(Equal(fakePass))
		Expect(mergedConfig.Storage.Files[0].Mode).To(Equal(swag.Int(0644)))
		Expect(mergedConfig.Storage.Filesystems[0].Device).To(Equal(fakeDevice))
		Expect(mergedConfig.Storage.Filesystems[0].Format).To(Equal(fakeFormat))
		Expect(mergedConfig.Storage.Filesystems[0].Path).To(Equal(fakePath))
	})
})

func TestIgnition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ignition_test")
}
