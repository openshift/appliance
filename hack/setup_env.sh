#!/bin/bash

yum install -y install podman genisoimage && yum clean all
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.25.0
GOFLAGS='' go install golang.org/x/tools/cmd/goimports@v0.1.5
GOFLAGS='' go install github.com/golang/mock/mockgen@v1.6.0
