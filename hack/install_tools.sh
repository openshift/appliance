#!/usr/bin/env bash

function test_tools() {
  echo "Installing test tools"
  which gocov 2>&1 >/dev/null || go install github.com/axw/gocov/gocov@v1.1.0
  which gocov-xml 2>&1 >/dev/null || go install github.com/AlekSi/gocov-xml@v1.1.0
}

test_tools
