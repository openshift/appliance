IMAGE := $(or ${IMAGE}, quay.io/edge-infrastructure/openshift-appliance:latest)
ISO_BUILDER_IMAGE := $(or ${ISO_BUILDER_IMAGE}, quay.io/edge-infrastructure/openshift-iso-builder:latest)
PWD = $(shell pwd)
LOG_LEVEL := $(or ${LOG_LEVEL}, info)
CMD := $(or ${CMD}, build)
ASSETS := $(or ${ASSETS}, $(PWD)/assets)

CI ?= false
ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
REPORTS ?= $(ROOT_DIR)/reports
COVER_PROFILE := $(or ${COVER_PROFILE},$(REPORTS)/unit_coverage.out)

CI ?= false
VERBOSE ?= false
GO_TEST_FORMAT = pkgname

GOTEST_FLAGS = --format=$(GO_TEST_FORMAT) $(GOTEST_PUBLISH_FLAGS) -- -count=1 -cover -coverprofile=$(REPORTS)/$(TEST_SCENARIO)_coverage.out
GINKGO_FLAGS = -ginkgo.focus="$(FOCUS)" -ginkgo.v -ginkgo.skip="$(SKIP)" -ginkgo.v -ginkgo.junit-report=./junit_$(TEST_SCENARIO)_test.xml

TIMEOUT = 30m
GINKGO_REPORTFILE := $(or $(GINKGO_REPORTFILE), ./junit_unit_test.xml)
# Required by go.podman.io/image: use pure-Go OpenPGP (no GPGME C dep) and skip the btrfs driver.
GO_BUILD_TAGS = containers_image_openpgp,exclude_graphdriver_btrfs
GO_UNITTEST_FLAGS = --format=$(GO_TEST_FORMAT) $(GOTEST_PUBLISH_FLAGS) -- -tags $(GO_BUILD_TAGS) -count=1 -cover -coverprofile=$(COVER_PROFILE)
GINKGO_UNITTEST_FLAGS = -ginkgo.focus="$(FOCUS)" -ginkgo.v -ginkgo.skip="$(SKIP)" -ginkgo.v -ginkgo.junit-report=$(GINKGO_REPORTFILE)


.PHONY: build

build:
	podman build -f Dockerfile.openshift-appliance . -t $(IMAGE)

build-appliance:
	mkdir -p build
	cd ./cmd && CGO_ENABLED=1 GOFLAGS="" go build -tags $(GO_BUILD_TAGS) -o ../build/openshift-appliance

build-iso-builder:
	mkdir -p build
	cd ./pkg/iso-builder && go generate ./...
	cd ./cmd/iso-builder && CGO_ENABLED=0 GOFLAGS="" go build -o ../../build/openshift-iso-builder

build-iso-config-embedder:
	mkdir -p build
	cd ./cmd/iso-config-embedder && CGO_ENABLED=0 GOFLAGS="" go build -o ../../build/iso-config-embedder

build-iso-builder-tools: build-iso-builder build-iso-config-embedder

build-iso-builder-image:
	podman build -f Dockerfile.iso-builder . -t $(ISO_BUILDER_IMAGE)

build-openshift-ci-test-bin:
	./hack/setup_env.sh

lint:
	golangci-lint run -v --timeout=20m --build-tags $(GO_BUILD_TAGS)

test: $(REPORTS)
	go test -tags $(GO_BUILD_TAGS) -v -count=1 -cover -coverprofile=$(COVER_PROFILE) ./...
	$(MAKE) _coverage

_coverage:
ifeq ($(CI), true)
	COVER_PROFILE=$(COVER_PROFILE) ./hack/publish-codecov.sh
endif

test-short:
	go test -tags $(GO_BUILD_TAGS) -short ./...

generate:
	go generate $(shell go list ./...)
	$(MAKE) format

format:
	@goimports -w -l main.go internal pkg || /bin/true

run: 
	podman run --rm -it \
		-v $(ASSETS):/assets:Z \
		--privileged \
		--net=host \
		$(IMAGE) $(CMD) --log-level $(LOG_LEVEL)

all: lint test build run

$(REPORTS):
	-mkdir -p $(REPORTS)

clean:
	-rm -rf $(REPORTS)

generate-mocks:
	find . -name 'mock_*.go' -type f -not -path './vendor/*' -delete
	go generate -v $(shell go list ./...)

unit-test:
	$(MAKE) _unit_test TIMEOUT=30m TEST="$(or $(TEST),$(shell go list ./...))"

_unit_test: $(REPORTS)
	# TODO: Add code coverage reports
	gotestsum $(GO_UNITTEST_FLAGS) $(TEST) $(GINKGO_UNITTEST_FLAGS) -timeout $(TIMEOUT)

.PHONY: integration-test-fast
integration-test-fast:
	go test -v -count=1 -short -tags integration ./tests/integration/...

.PHONY: integration-test
integration-test:
	go test -count=1 -timeout $(TIMEOUT) -tags integration ./tests/integration/...

update-rpm-lockfile:
	./hack/update-rpm-lockfile.sh
