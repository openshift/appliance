# pkg/iso-builder/config

This package defines the iso-builder configuration model and the binary embed
mechanism used to carry that configuration inside a compiled binary.

It is a **standalone Go module** (`github.com/openshift/appliance/pkg/iso-builder/config`)
with no dependencies outside the standard library, so external tools can import
it without pulling the full appliance dependency tree.

## What it contains

### Configuration model (`config.go`)

`Config` is the top-level struct that describes an OpenShift installation ISO.
Fields follow the OpenShift installer naming conventions and are serialised as
JSON/YAML. Supporting types: `Proxy`, `OLMOperator`, `ExtraManifest`.

### Binary embed area (`embed.go`)

The iso-builder binary reserves a 1 MiB area delimited by start/end markers.
The `iso-config-embedder` tool writes a base64-encoded JSON config into this
area after compilation; at runtime the binary reads it back.

Key functions:

| Function | Purpose |
|---|---|
| `Encode` / `Decode` | Config <-> base64-encoded JSON |
| `ReadFromData` / `ReadFromBinary` | Extract config from raw bytes or a file |
| `WriteToData` / `WriteToBinary` | Write config into the embed area of raw bytes or a file |

The markers (`ConfigStartMarker`, `ConfigEndMarker`) and area size
(`ConfigEmbedSize`) are exported so that test helpers and the embed-area
generator can construct valid fake binaries.
