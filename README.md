# OpenShift-based Appliance Builder

`openshift-appliance` is a command line utility for building a disk image that orchestrates OpenShift installation using the Agent-based installer. 
The primary use-case of the appliance is to support a fully disconnected installation
of an OpenShift cluster. Thus, all required images are included in the appliance disk image.

## Quick Start

Build and run: binary or container image

### Using the binary
* Install dependencies (libguestfs-tools/coreos-installer/oc)
* make build-appliance
* ./bin/openshift-appliance

#### Commands
* build
* clean
* generate-config

#### Flags
* --dir
* --log-level

### Using a container image

#### Configuration
``` bash
export IMAGE=<image_url>
export ASSETS=<assets_dir>
export LOG_LEVEL=info/debug/error
export CMD=build/clean/generate-config
```

#### Build and Run
make build run

## Development

### Running tests
```bash
skipper make test
```

### Running lint
```bash
make lint
```

### Debug

#### Bootstrap step

Add `--debug-bootstrap` flag to the build command to avoid machine reboot on bootstrap step completion. Useful for taking a snapshot of the appliance disk image before testing changes in the install ignition. 

#### Install step

Add `--debug-install` flag to the build command for enabling ssh login on the installation step.
The public ssh key provided in appliance-config.yaml is used (`sshKey` property).

#### Test changes in the install ignition

To debug/test changes made in the `InstallIgnition` asset, follow the steps described on [test_install_ignition.md](/hack/diskimage/test_install_ignition.md)

### Cleanup

After a successful build, use the `clean` command before re-building the appliance (removes temp folder and state file).

## Main Components

### Recovery ISO Assets (pkg/asset/recovery/)
* BaseISO - a CoreOS LiveCD used as a base for the recovery ISO
* RecoveryISO - the base ISO with an embedded recovery ignition

### Appliance Assets (pkg/asset/appliance/)
* BaseDiskImage - a CoreOS qcow2 disk image used as a base for the appliance disk image
* ApplianceDiskImage - the output disk image of the builder

### Ignition Assets (pkg/asset/ignition/)
* BootstrapIgnition - the ignition config used for cluster bootstrap step
* InstallIgnition - the ignition config used for cluster installation step
* RecoveryIgnition - the ignition config embedded into the recovery ISO

## High-Level Flow
