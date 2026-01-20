package utils

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-openapi/swag"
	"github.com/openshift/appliance/pkg/asset/config"
	"github.com/openshift/appliance/pkg/types"
)

  func TestUtils(t *testing.T) {
      RegisterFailHandler(Fail)
      RunSpecs(t, "Utils Suite")
  }

  var _ = Describe("ExtractVersionFromURL", func() {
      DescribeTable("should extract version from URL",
          func(url string, expectedResult string) {
              result := extractVersionFromURL(url)
              Expect(result).To(Equal(expectedResult))
          },
          Entry("standard OCP stable release URL",
              "quay.io/openshift-release-dev/ocp-release:4.20.4-x86_64",
              "4.20.4-x86_64",
          ),
          Entry("URL with registry port",
              "localhost:5000/ocp-release:4.20.4-x86_64",
              "4.20.4-x86_64",
          ),
      )
  })

  var _ = Describe("GetOCPVersion", func() {
      DescribeTable("should get OCP version",
          func(applianceConfig *config.ApplianceConfig, expectedResult string) {
              result := GetOCPVersion(applianceConfig)
              Expect(result).To(Equal(expectedResult))
          },
          Entry("extract version from URL",
              &config.ApplianceConfig{
                  Config: &types.ApplianceConfig{
                      OcpRelease: types.ReleaseImage{
                          Version:         "4.20.4",
                          CpuArchitecture: swag.String("x86_64"),
                          URL:             swag.String("quay.io/openshift-release-dev/ocp-release:4.20.4-x86_64"),
                      },
                  },
              },
              "4.20.4-x86_64",
          ),
          Entry("extract version from URL with registry port",
              &config.ApplianceConfig{
                  Config: &types.ApplianceConfig{
                      OcpRelease: types.ReleaseImage{
                          Version:         "4.20.4",
                          CpuArchitecture: swag.String("x86_64"),
                          URL:             swag.String("localhost:5000/openshift-release:4.20.4-x86_64"),
                      },
                  },
              },
              "4.20.4-x86_64",
          ),
          Entry("construct version from version and architecture",
              &config.ApplianceConfig{
                  Config: &types.ApplianceConfig{
                      OcpRelease: types.ReleaseImage{
                          Version:         "4.20.4",
                          CpuArchitecture: swag.String("x86_64"),
                      },
                  },
              },
              "4.20.4-x86_64",
          ),
      )
  })
