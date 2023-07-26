package graph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-openapi/swag"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

const (
	cincinnatiPartialResponse = "quay.io/openshift-release-dev/ocp-release@sha256"
	fakeCincinnatiAddress     = "https://api.example.com/api/upgrades_info/graph"
)

type ClientMock struct{}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), cincinnatiAddress) {
		cincinnatiFakeResponse := Response{
			Nodes: []Release{
				{
					Version: "4.13.1",
					Payload: fmt.Sprintf("%s:foobar1", cincinnatiPartialResponse),
				},
				{
					Version: "4.13.2",
					Payload: fmt.Sprintf("%s:foobar2", cincinnatiPartialResponse),
				},
				{
					Version: "4.13.3",
					Payload: fmt.Sprintf("%s:foobar3", cincinnatiPartialResponse),
				},
			},
		}

		responseJSON, err := json.Marshal(cincinnatiFakeResponse)
		Expect(err).ToNot(HaveOccurred())

		response := &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			ContentLength: int64(len(responseJSON)),
			Body:          io.NopCloser(bytes.NewReader(responseJSON)),
		}

		return response, nil
	} else if strings.Contains(req.URL.String(), fakeCincinnatiAddress) {
		response := &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}

		return response, nil
	}

	return nil, errors.New("test client error, unexpected URL")
}

var _ = Describe("Test Graph", func() {
	var (
		testGraph   Graph
		graphConfig GraphConfig
	)

	BeforeEach(func() {
		channel := ReleaseChannelStable
		graphConfig = GraphConfig{
			HTTPClient: &ClientMock{},
			Arch:       "amd64",
			Channel:    &channel,
		}
	})

	It("GetReleaseImage - Valid version", func() {
		version := "4.13.1"
		graphConfig.Version = version

		testGraph = NewGraph(graphConfig)
		cincinnatiResponse, verResponse, err := testGraph.GetReleaseImage()
		Expect(err).ToNot(HaveOccurred())
		Expect(cincinnatiResponse).To(ContainSubstring(cincinnatiPartialResponse))
		Expect(verResponse).To(Equal(version))
	})

	It("GetReleaseImage - Invalid version", func() {
		graphConfig.Version = "4.13.100"

		testGraph = NewGraph(graphConfig)
		_, _, err := testGraph.GetReleaseImage()
		Expect(err).To(HaveOccurred())
	})

	It("GetReleaseImage - Cincinnati returns 404", func() {
		graphConfig.Version = "4.13.1"
		graphConfig.CincinnatiAddress = swag.String(fakeCincinnatiAddress)

		testGraph = NewGraph(graphConfig)
		_, _, err := testGraph.GetReleaseImage()
		Expect(err).To(HaveOccurred())
	})
})

func TestGraph(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "graph_test")
}
