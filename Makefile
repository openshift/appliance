IMAGE := $(or ${IMAGE}, quay.io/derez/openshift-appliance:latest)
PWD = $(shell pwd)
LOG_LEVEL := $(or ${LOG_LEVEL}, info)
CMD := $(or ${CMD}, build)
ASSETS := $(or ${ASSETS}, $(PWD)/assets)

CI ?= false
ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
REPORTS ?= $(ROOT_DIR)/reports
COVER_PROFILE := $(or ${COVER_PROFILE},$(REPORTS)/unit_coverage.out)

.PHONY: build

build:
	podman build -f Dockerfile.openshift-appliance . -t $(IMAGE)

build-appliance:
	mkdir -p build
	cd ./cmd && CGO_ENABLED=0 GOFLAGS="" GO111MODULE=on go build -o ../build/openshift-appliance

build-openshift-ci-test-bin:
	./hack/setup_env.sh

lint:
	golangci-lint run -v --timeout=5m

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
		$(IMAGE) $(CMD) --log-level $(LOG_LEVEL)

all: lint test build run

$(REPORTS):
	-mkdir -p $(REPORTS)

clean:
	-rm -rf $(REPORTS)
