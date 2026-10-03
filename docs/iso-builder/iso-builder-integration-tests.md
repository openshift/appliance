# Integration tests

## Definition

A unit test usually calls Go functions or packages directly, focusing on a small portion of the project's code. 
Dependencies are often mocked or isolated.

An integration test exercises the CLI as a real executable and verifies that multiple components work together
as expected, including argument parsing, command logic, filesystem interactions, configuration loading, output, 
and exit codes.

From this perspective, unit tests are typically smaller and faster than integration tests, and a project usually
contains many more unit tests than integration tests.

## Test framework

Use `github.com/rogpeppe/go-internal/testscript` for CLI integration tests.

## Test organization

Define test scenarios in `.txt` files under `testdata`, and use them to execute the real CLI, set up files and
environment variables, and assert exit codes, stdout, stderr, and filesystem changes.

Keep each script focused on a single user-visible CLI behavior or scenario.

Use the following structure:

tests/
└── integration/
    ├── integration_test.go
    └── testdata/
        ├── basic.txt
        ├── config.txt
        └── errors.txt

Create a makefile target `make integration-tests` to run only integration tests. Make sure that other makefile targets
do not run the integration tests

## Tests

### build: missing configuration

* setup: just the iso-builder, with no config
* run: openshift-iso-builder build
* verify: error due missing config

### show-config: missing configuration

* setup: just the iso-builder, with no config
* run: openshift-iso-builder show-config
* verify: error due missing config
