# iso-builder CLI — Design Specification

## 1. Overview

The iso-builder is a CLI tool that allows users to build, locally on their systems, an installation
ISO for disconnected environments, containing a specific OCP release and a custom set of OLM
operators.

### 1.1 Motivations

This approach was specifically designed to simplify air-gapped cluster deployments: it does not
require the presence of an external registry, and it offers a local installation UI. For more
technical details about this method see [Simplified registry-less cluster operations for disconnected environments](https://github.com/openshift/enhancements/blob/master/enhancements/agent-installer/simplified-registry-less-cluster-operations.md).

To preserve ease of use, the iso-builder exposes a minimal interface built around a 
fixed-configuration model, and is distributed as a single executable file.

## 2. User workflow

The first step consists of downloading the iso-builder tool from the 
[Assisted Installer web console](https://console.redhat.com/openshift/assisted-installer/clusters/~new)
page for disconnected installations, where the user can select a specific OCP version and a
custom set of OLM operators to install. In addition, a limited set of common install configuration
can be optionally provided (this part of the workflow is not in scope for the current document).

Note that the retrieved binary will embed the selected configuration chosen in the web console, and
it cannot be further modified once downloaded. This means that each specific downloaded instance
of the iso-builder is immutable, and it will produce the same ISO when invoked (with a notable
exception for custom operators, see later). 
If the user wants to generate a different ISO, a new iso-builder instance with a different
basic configuration (ie, a different OCP version, or a different set of OLM operators) must be
retrieved from the Assisted Installer web console.

Once downloaded, the iso-builder must be executed either in a connected environment or in an
environment containing a registry with the required mirrored data to successfully produce the ISO.
The generated ISO can then be tested by the customer, and then moved to the target disconnected
environment for the effective installation.

### 2.1 Usage Examples

```bash
# Example: build an ISO in the current directory
$ openshift-iso-builder build

# Example: build an ISO in the specified directory
$ openshift-iso-builder build --working-dir /tmp

# Example: show the current builder configuration
$ openshift-iso-builder show-config
```

## 3. Commands

| Command     | Description                                    | Output                         |
|-------------|------------------------------------------------|--------------------------------|
| build       | Build the ISO using the embedded configuration | agent.\<arch\>.iso             |
| show-config | Displays, in a human-readable way, the current | Summary of the embedded config |
|             | embedded configuration                         |                                |

### 3.1 Build command flags

| Flag                     | Default             | Description                                                                                   |
|--------------------------|---------------------|-----------------------------------------------------------------------------------------------|
| `--working-dir`          | current working dir | Set the working directory                                                                     |
| `--additional-image`     | -                   | Set an additional (application) image to be added in the ISO. Can be specified multiple times |

### 3.2 Global Flags

| Flag          | Default | Description               |
|---------------|---------|---------------------------|
| `--log-level` | `info`  | Logging verbosity         |
| `--quiet`     | `false` | Suppress non-error output |

## 4. Architecture & Key Decisions

A key factor in the iso-builder architecture is its immutability model. This is achieved
by embedding a JSON config file into the executable. So it is required
to create an embedding area of at least 1MiB with proper markers within the executable to store 
the config file. It should be possible to use a similar technique already adopted by oc to overlay
data into the openshift-installer.

### 4.1 iso-builder config file format

The config file must contain the following fields:

| Field                   | Type          | Description                                                                |
|-------------------------|---------------|----------------------------------------------------------------------------|
| `openshiftVersion`      | `string`      | OCP version to be mirrored                                                 |
| `releaseImageURL`       | `string`      | pullspec for the release image. To be used instead of `openshiftVersion`   |
| `pullSecret`            | `string`      | Secret to use when pulling images                                          |
| `sshKey`                | `string list` | Public Secure Shell (SSH) key to provide access to instances               |
| `architecture`          | `string`      | Cluster CPU architecture                                                   |
| `proxy`                 | `object`      | Settings for the cluster proxy (http/https/noProxy)                        |
| `additionalTrustBundle` | `string`      | PEM-encoded X.509 certificate bundle for nodes trusted certificate store   |
| `additionalNTPServers`  | `string list` | List of additional NTP servers to use for provisioning                     |
| `fips`                  | `boolean`     | Configures https://www.nist.gov/itl/fips-general-information               |
| `olmOperators`          | `object list` | OLM operators to be mirrored (name/version/channel)                        |   
| `rendezvousIP`          | `string`      | Defines the rendezvous IP                                                  |
| `networkConfig`         | `object list` | Nodes NMState config to define static networking                           |
| `additionalImages`      | `string list` | Additional images pullspec to be mirrored                                  |
| `extraManifests`        | `object list` | Additional custom manifests to be added in the cluster                     |

Follow the openshift installer naming convention for the fields (see https://github.com/openshift/installer/blob/main/data/data/install.openshift.io_installconfigs.yaml)

#### 4.1.1 config file format export 

The config file must be defined in its own go module, to facilitate importing into other go projects
such as assisted-service without requiring excessive dependency resolution.
Include also the method to read / write the config file in the binary here.

### 4.2 Relationship to existing appliance code

The goal for the current prototype is to reuse as much as possible the existing appliance code.
For this reason initially the embedded config needs to be converted to the configuration format used by
the existing appliance code, to run the ISO build.

### 4.3 Internal structure

```
cmd/
├── iso-builder/                  # main() for the iso-builder CLI
└── iso-config-embedder/          # main() for the iso-config-embedder CLI

pkg/
├── iso-builder/                  # Root: CLI wiring (cobra root command + subcommands)
│   ├── commands/                 # Command implementations (Builder, ShowConfig)
│   ├── config/                   # Config model and binary embed I/O (standalone module)
│   ├── embeddedconfig/           # Runtime loader for the embedded config area
│   └── gen_embed_area/           # Build-time generator for config_embed_area.bin
└── iso-config-embedder/          # iso-config-embedder tool (embed + show commands)
```

| Package | Role |
|---|---|
| `pkg/iso-builder` | Defines the iso-builder CLI command tree (root, `build`, `show-config`) via cobra. Entry point: `Run()`. |
| `pkg/iso-builder/commands` | Implements the iso-builder commands: `Builder` orchestrates the ISO build, `ShowConfig` displays the embedded configuration. |
| `pkg/iso-builder/config` | Standalone Go module defining the `Config` model, serialisation (JSON/base64), and binary embed read/write. Importable by external projects (e.g. assisted-service) without pulling the full dependency tree. |
| `pkg/iso-builder/embeddedconfig` | Holds the `config_embed_area.bin` blob (`//go:embed`), its generator directive (`//go:generate`), and the `LoadConfig()` function that decodes the embedded area at runtime. Keeps `rawConfigArea` unexported. |
| `pkg/iso-builder/gen_embed_area` | Build-time generator (`go generate`) that produces `config_embed_area.bin` (start marker + 1 MiB NUL pad + end marker). |
| `pkg/iso-config-embedder` | Implements the iso-config-embedder CLI, a separate tool that embeds a YAML config into an iso-builder binary and can read it back. Entry point: `Run()`. |

### 4.4 External dependencies

The long term goal is to remove all the external dependencies. We'd like to support both linux and
macOS environment, so it's fundamental that the iso-builder will remain a single executable file,
without any other requirements. The only exceptions will be provided by the other official Red Hat
tool, such as oc-mirror and oc: it will not be possible to embed them, so they will be downloaded
as a first step.

### 4.5 Testing tool

Create a separate testing binary that can take a config file in YAML format, convert it to JSON and
embed it in the ISO builder binary. It should also be able to read the embedded config, convert it
back to YAML and dump it to stdout. Include this binary in the same container image as the ISO
builder binary.

The reading and writing routines should be implemented in the same go module as the config file
format, so that they can be reused by assisted-image-service. We will use this tool to stand in for
assisted-service in CI testing.

### 4.6 Build & distribution

Until we'll keep the old appliance code in place, let's share the same Makefile to create new
specific iso-builder targets.
* Add a target to build the binary (mostly used for local development/testing)
* Add a target to build the testing binary (mostly used for local development/testing)
* Add a target to build the binary in a Dockerfile, using an ubi-micro image: it's just for 
  eventual distribution, as we do not expect to run the tool from within the container

## 5. Iso-builder steps

When the user runs `openshift-iso-builder build`, the `Builder` in
`pkg/iso-builder/commands/` executes the following steps in order:

1. Extract the embedded configuration from the binary.
2. Set up the user cache. All artifacts that are invariant across multiple runs for the
same OCP release are stored in a per-release cache under `~/.cache/iso-builder/release-<version>/`.
The cache is created on first use and reused on subsequent builds.
3. Download tools such as `oc` and `oc-mirror` into the cache. Each download is idempotent.
4. Check release availability. Verify that the release image is reachable, either from
   quay.io for connected environemnt or from the user provided local registry for disconnected environment.
5. Extract openshift installer binary from the release image (FIPS variant when enabled). This download is also idempotent.
6. Finally, generate the live ISO by delegating to the existing appliance builder.

## 6. Artifact Organization

### 6.1 User-level cache (shared across builds)

All artifacts that are **invariant** across multiple iso-builder runs for the same OCP
release version are stored under `~/.cache/iso-builder/`.

```
~/.cache/iso-builder/
└── release-<version>/                 # e.g. release-4.22.18
    ├── tools/
    │   ├── oc
    │   ├── oc-mirror
    │   └── openshift-install
    ├── images/
    │   └── <mirrored container images managed by oc-mirror>
    ├── registry/
    │   └── <registry data produced by the legacy builder>
    └── iso/
        └── <base CoreOS ISO and other release-level ISO artifacts>
```

| Directory | Contents | Shared? |
|-----------|----------|---------|
| `tools/`  | `oc`, `oc-mirror`, `openshift-install` binaries | Yes — same binaries for every build of this release |
| `images/` | Container images mirrored by `oc-mirror` | Yes — same release images regardless of cluster topology |
| `registry/` | Registry data produced during the build | Yes — same registry content for a given release |
| `iso/`    | CoreOS base ISO and similar release-level artifacts | Yes — same base ISO for every build of this release |

**Key**: `<version>` is the full OCP version string (e.g. `4.22.18`, `4.23.0`, `5.0.0`).
Multiple releases coexist side by side. For example, running builds for both `4.22.18`
and `4.22.17` produces:

```
~/.cache/iso-builder/
├── release-4.22.17/
│   ├── tools/
│   ├── images/
│   ├── registry/
│   └── iso/
└── release-4.22.18/
    ├── tools/
    ├── images/
    ├── registry/
    └── iso/
```

### 6.2 Working directory (per-build, transient)

All artifacts specific to a particular build session are written to the working
directory. The working directory is set by `--working-dir` (default: current directory).

```
<working-dir>/
├── agent.x86_64.iso                   # final output ISO
└── .iso-builder/                      # transient build artifacts
    └── <appliance build state>
```

| Path | Contents |
|------|----------|
| `agent.<arch>.iso` | The final output live ISO |
| `.iso-builder/` | Internal build state generated by the appliance code. Includes intermediate ignition files, temp registry data, and other transient artifacts. This directory can be safely deleted after a successful build |

### 6.3 Multi-build coexistence examples

Given these user invocations:

```bash
iso-builder-4.22.18-sno        build --working-dir ~/builds/sno
iso-builder-4.22.18-sno        build --working-dir ~/builds/sno-custom \
                                     --additional-image my-super-op \
                                     --additional-image another-op
iso-builder-4.22.18-ha5        build --working-dir ~/builds/ha5
iso-builder-4.22.17-compact    build --working-dir ~/builds/compact-17
iso-builder-5.0-compact        build --working-dir ~/builds/compact-50
iso-builder-4.23-ha-fips       build --working-dir ~/builds/ha-fips
```

The resulting disk layout is:

```
~/.cache/iso-builder/
├── release-4.22.17/
│   ├── tools/   {oc, oc-mirror, openshift-install}
│   ├── images/
│   ├── registry/
│   └── iso/
├── release-4.22.18/               ← shared by sno, sno-custom, ha5
│   ├── tools/   {oc, oc-mirror, openshift-install}
│   ├── images/
│   ├── registry/
│   └── iso/
├── release-4.23.0/
│   ├── tools/   {oc, oc-mirror, openshift-install-fips}
│   ├── images/
│   ├── registry/
│   └── iso/
└── release-5.0.0/
    ├── tools/   {oc, oc-mirror, openshift-install}
    ├── images/
    ├── registry/
    └── iso/

~/builds/
├── sno/
│   ├── agent.x86_64.iso
│   └── .iso-builder/
├── sno-custom/
│   ├── agent.x86_64.iso
│   └── .iso-builder/
├── ha5/
│   ├── agent.x86_64.iso
│   └── .iso-builder/
├── compact-17/
│   ├── agent.x86_64.iso
│   └── .iso-builder/
├── compact-50/
│   ├── agent.x86_64.iso
│   └── .iso-builder/
└── ha-fips/
    ├── agent.x86_64.iso
    └── .iso-builder/
```

The three `4.22.18` builds (sno, sno-custom, ha5) share a single set of cached tools,
images, and registry data. Each build produces its own output ISO and transient state
in its respective working directory.

## 7. Constraints

* CLI output must be human-readable, not machine-parseable
* CLI output/logs must not contain in any case sensitive information like the pull-secret
* The embedded config must be encoded in base64, and the user must not be able to modify it
* The overall size of the compiled executable must remain small, less than 150MiB
* The CLI UX (given by the exposed set of commands/flags) must remain as much as possible
  minimalistic
* The cache location (`~/.cache/iso-builder/`) is not configurable in the initial
  implementation. A `--cache-dir` flag may be added later.
* Only the connected (quay.io) use case is supported for `checkReleaseAvailability`
  in this iteration. Disconnected (local registry) support is a future enhancement.
* The legacy `applyLiveISOBuilderAsset` is not modified. The new pipeline only
  prepares the environment and delegates to it.

## 8. Non-Goals

* Will not address the SaaS UI development
* Modifying the legacy appliance asset pipeline
* Implementing the disconnected (local registry) path for release availability
* Adding a `--cache-dir` CLI flag
* Garbage collection or eviction of old cache entries

## 9. JIRA references

* https://redhat.atlassian.net/browse/AGENT-1592