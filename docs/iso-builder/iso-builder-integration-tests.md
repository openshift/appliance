# Integration tests

## Definition

A unit test usually calls Go functions or packages directly, focusing on a small portion of the project's code. 
Internal dependencies are often mocked or isolated.

An integration test exercises the CLI as a real executable and verifies that multiple components work together
as expected, including argument parsing, command logic, filesystem interactions, configuration loading, output, 
and exit codes. External complex dependencies can be mocked.

An e2e test uses the effective CLI executable in a real world scenario, addressing in particular customer-oriented
scenarios. Usually no mocks are used at all.

From this perspective, unit tests are typically smaller and faster than integration tests, and a project usually
contains many more unit tests than integration tests. e2e tests can also be fewer, due to their increased complexity
and execution time.

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

Some tests could be pretty quick to execute, while other could take several minutes to complete:
* create a makefile target `make integration-test-fast` to run only the fast integration tests (make sure that other makefile targets
do not run the integration tests)
* create a makefile target `make integration-test` to run all the integration test. Do not run them if not explicitly required.

---

## Tests

### build 

#### missing configuration
* setup: just the iso-builder, with no config
* run: build command
* verify: error due to missing config

#### missing ocp version
* setup: minimal config but without ocp version
* run: build command
* verify: error due to missing ocp version

#### missing pull-secret
* setup: minimal config but without a pull-secret
* run: build command
* verify: error due to missing pull-secret

#### (long) basic config: okd payload
* setup: 
  - create a minimal config just using quay.io/okd/scos-release:5.0.0-okd-scos.0. 
    Since it does not require a pull secret, use a dummy one
  - embed the config in the iso-builder
* run: build command
* verify: iso with the OCP payload is generated

### show-config

#### missing configuration
* setup: just the iso-builder, with no config
* run: show-config command
* verify: error due to missing config

#### complete sample config
* setup: 
  - create a sample but complete embedder yaml config, containing all the fields
  - embed the config in the iso-builder
* run: show-config command
* verify: 
  - the embedder succeeded
  - the content shown matches the yaml config

