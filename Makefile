IMAGE := $(or ${IMAGE}, quay.io/edge-infrastructure/openshift-appliance:latest)
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
GINKGO_FLAGS = -ginkgo.focus="$(FOCUS)" -ginkgo.v -ginkgo.skip="$(SKIP)" -ginkgo.v -ginkgo.reportFile=./junit_$(TEST_SCENARIO)_test.xml

TIMEOUT = 30m
GINKGO_REPORTFILE := $(or $(GINKGO_REPORTFILE), ./junit_unit_test.xml)
GO_UNITTEST_FLAGS = --format=$(GO_TEST_FORMAT) $(GOTEST_PUBLISH_FLAGS) -- -count=1 -cover -coverprofile=$(COVER_PROFILE)
GINKGO_UNITTEST_FLAGS = -ginkgo.focus="$(FOCUS)" -ginkgo.v -ginkgo.skip="$(SKIP)" -ginkgo.v -ginkgo.reportFile=$(GINKGO_REPORTFILE)


.PHONY: build

build:
	podman build -f Dockerfile.openshift-appliance . -t $(IMAGE)

build-appliance:
	mkdir -p build
	cd ./cmd && CGO_ENABLED=0 GOFLAGS="" GO111MODULE=on go build -o ../build/openshift-appliance

build-openshift-ci-test-bin:
	./hack/setup_env.sh

lint:
	golangci-lint run -v --timeout=10m

test: $(REPORTS)
	go test -count=1 -cover -coverprofile=$(COVER_PROFILE) ./...
	$(MAKE) _coverage

_coverage:
ifeq ($(CI), true)
	COVER_PROFILE=$(COVER_PROFILE) ./hack/publish-codecov.sh
endif

test-short:
	go test -short ./...

test-integration:
	go test ./integration_test/...

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
	gotestsum $(GO_UNITTEST_FLAGS) $(TEST) $(GINKGO_UNITTEST_FLAGS) -timeout $(TIMEOUT) || ($(MAKE) _post_unit_test && /bin/false)
	$(MAKE) _post_unit_test

_post_unit_test: $(REPORTS)
	@for name in `find '$(ROOT_DIR)' -name 'junit*.xml' -type f -not -path '$(REPORTS)/*'`; do \
		mv -f $$name $(REPORTS)/junit_unit_$$(basename $$(dirname $$name)).xml; \
	done
	$(MAKE) _unit_test_coverage

_unit_test_coverage: $(REPORTS)
ifeq ($(CI), true)
	gocov convert $(REPORTS)/unit_coverage.out | gocov-xml > $(REPORTS)/unit_coverage.xml
	./hack/publish-codecov.sh
endif
