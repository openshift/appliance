# iso-config-embedder

A testing and CI companion tool for `openshift-iso-builder`. It embeds a YAML configuration file
into the iso-builder binary and can show an embedded configuration for inspection.

In production, the configuration is embedded by assisted-images-service when the user downloads the
iso-builder from the Assisted Installer web console. This tool stands in for assisted-image-service
during development and CI testing.

## Build

```bash
make build-iso-config-embedder
```

The tool is also built as part of the `Dockerfile.iso-builder` container image.

## Usage

### Embed a configuration into the iso-builder binary

```
Usage: iso-config-embedder embed [OPTIONS] <BINARY>

Arguments:
  <BINARY>  iso-builder binary (source, not modified)

Options:
  -o, --output <path>  Write the resulting binary to this path (required)
  -c, --config <path>  Path to the YAML config file [default: stdin]
  -f, --force          Overwrite an existing embedded config
```

Examples:

```bash
# Embed from a file
iso-config-embedder embed -c config.yaml -o patched-iso-builder openshift-iso-builder

# Embed from stdin
cat config.yaml | iso-config-embedder embed -o patched-iso-builder openshift-iso-builder
```

The YAML file is validated strictly: unknown fields are rejected.

### Show the embedded configuration

```
Usage: iso-config-embedder show <BINARY>

Arguments:
  <BINARY>  iso-builder binary
```

Prints the embedded configuration as YAML to stdout.

## Config file format

The only required field is `pullSecret`. All other fields are optional.

| Field                   | Type            | Description                                                            |
|-------------------------|-----------------|------------------------------------------------------------------------|
| `pullSecret`            | string          | Secret to use when pulling images (**required**)                       |
| `openshiftVersion`      | string          | OCP version to be mirrored                                             |
| `releaseImageURL`       | string          | Pullspec for the release image; used instead of `openshiftVersion`     |
| `sshKey`                | string list     | Public SSH keys to provide access to instances                         |
| `architecture`          | string          | Cluster CPU architecture (e.g. `x86_64`, `aarch64`)                   |
| `proxy`                 | object          | Cluster proxy settings (see below)                                     |
| `additionalTrustBundle` | string          | PEM-encoded X.509 certificate bundle for the nodes trust store         |
| `additionalNTPServers`  | string list     | Additional NTP servers to use for provisioning                         |
| `fips`                  | boolean         | Enables FIPS mode                                                      |
| `olmOperators`          | object list     | OLM operators to be mirrored (see below)                               |
| `rendezvousIP`          | string          | IP address used as rendezvous point                                    |
| `networkConfig`         | object list     | NMState configs for static networking (opaque YAML objects)            |
| `additionalImages`      | string list     | Additional image pullspecs to be mirrored                              |
| `extraManifests`        | object list     | Custom Kubernetes manifests to include in the cluster (see below)      |

### proxy

| Field        | Type   | Description                                              |
|--------------|--------|----------------------------------------------------------|
| `httpProxy`  | string | HTTP proxy URL                                           |
| `httpsProxy` | string | HTTPS proxy URL                                          |
| `noProxy`    | string | Comma-separated list of destinations that bypass the proxy |

### olmOperators

| Field     | Type   | Description            |
|-----------|--------|------------------------|
| `name`    | string | Operator package name  |
| `version` | string | Operator version       |
| `channel` | string | OLM channel to track   |

### extraManifests

| Field     | Type   | Description          |
|-----------|--------|----------------------|
| `name`    | string | Manifest file name   |
| `content` | string | Raw manifest content |

## Example

A minimal configuration:

```yaml
pullSecret: '{"auths":{"cloud.openshift.com":{"auth":"..."}}}'
```

A full configuration:

```yaml
openshiftVersion: "4.22"
releaseImageURL: "quay.io/openshift-release-dev/ocp-release:4.22.0-x86_64"
pullSecret: '{"auths":{"cloud.openshift.com":{"auth":"..."}}}'
sshKey:
  - ssh-rsa AAAA...
architecture: x86_64
proxy:
  httpProxy: http://proxy.example.com:8080
  httpsProxy: https://proxy.example.com:8443
  noProxy: localhost,127.0.0.1,.example.com
additionalTrustBundle: |
  -----BEGIN CERTIFICATE-----
  MIID...
  -----END CERTIFICATE-----
additionalNTPServers:
  - ntp1.example.com
  - ntp2.example.com
fips: true
olmOperators:
  - name: local-storage-operator
    version: "4.22"
    channel: stable
rendezvousIP: 192.168.1.10
networkConfig:
  - interfaces:
      - name: eth0
        type: ethernet
        state: up
additionalImages:
  - registry.example.com/app:v1
extraManifests:
  - name: custom-configmap.yaml
    content: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: custom
      data:
        key: value
```
