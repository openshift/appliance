# CLAUDE.md — iso-builder

This file provides context for Claude Code when working on the **iso-builder** CLI tool.

## Project focus

We are building a new CLI tool called `iso-builder` within this repository.
The existing appliance code is a starting base — not the primary focus.
Refer to `docs/iso-builder/iso-builder-spec.md` for the full design specification and
`docs/iso-builder/acceptance/` for acceptance criteria.

## Code conventions

### Language & module

- Go (module: `github.com/openshift/appliance`)
- Follow standard Go conventions: `gofmt`, `go vet`, effective Go idioms

### Style

- No license headers in source files
- Imports grouped in three blocks separated by blank lines: stdlib, external, internal
- Use `github.com/pkg/errors` for error wrapping (existing convention in this repo)
- Use `github.com/sirupsen/logrus` for logging
- Use `github.com/spf13/cobra` for CLI commands
- Interfaces defined next to their consumer, not their implementation
- Do not invent behavior that is not covered by the acceptance criteria
- Ensure that any new portion of code is covered at least by one or more unit tests
- Avoid duplications, prefer a coding style that improves the readability and maintenance
- Do not perform broad refactors unless needed to make the behavior testable.
- Minimize comments, and keep them short.
- Ensure that any new portion of code is covered at least by one or more unit tests.
- Ensure to add a brief doc for each export symbol

### Naming

- Packages: short, lowercase, single-word when possible
- Files: lowercase with underscores (`my_thing.go`), tests as `my_thing_test.go`
- Exported types/functions: descriptive, no stuttering (`config.Load`, not `config.LoadConfig`)

### Testing

- `make lint` — runs `golangci-lint`; run before committing
- `make test` — default command to verify both legacy appliance code and new iso-builder
- `make unit-test` — runs tests via `gotestsum` with JUnit reporting; supports `TEST=./pkg/foo` to scope
- Test framework: golang
- Unit test files live next to the code they test
- When testing different cases for the same scenario, use the cases := []struct{} to capture the
  relevant key fields for the test. Add always a speaking name field to represent the current case.
- Reuse the existing test methods if possible.
- Keep the test focused on the behaviors, avoid testing unnecessary technical details.
- Do not add too many comments to the tests
- Try to use only the exported methods for a given type in the tests. 

### Error handling

- Wrap errors with context: `errors.Wrap(err, "descriptive message")`
- Fail early, return errors up — avoid silent swallowing
- CLI commands should log fatal errors via `logrus.Fatal` at the command level only
