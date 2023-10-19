# OpenShift Appliance User Guide

## What is OpenShift Appliance
The `openshift-appliance` utility enables self-contained OpenShift cluster installations, meaning that it does not rely on Internet connectivity or external registries.
It is a container-based utility that builds a disk image that includes the [Agent-based installer](https://cloud.redhat.com/blog/meet-the-new-agent-based-openshift-installer-1).
That disk image can then be used to install multiple OpenShift clusters.

## Download
OpenShift Appliance is available for download at: https://quay.io/edge-infrastructure/openshift-appliance

## High-Level Flow Overview

![hl-overview.png](images%2Fhl-overview.png)

### Lab
* This is where `openshift-appliance` gets used to create a raw sparse disk image.
  * **raw:** so it can be copied as-is to multiple servers. 
  * **sparse:** to minimize the physical size.
* The end result is a generic disk image with a partition layout as follows:
  ``` bash
  Name       Type        VFS      Label       Size  Parent
  /dev/sda2  filesystem  vfat     EFI-SYSTEM  127M  -
  /dev/sda3  filesystem  ext4     boot        350M  -
  /dev/sda4  filesystem  xfs      root        180G  -
  /dev/sda5  filesystem  ext4     agentboot   1.2G  -
  /dev/sda6  filesystem  iso9660  agentdata   18G   -
  ```
* The two additional partitions:
  * `agentboot`: Agent-based installer ISO:
    * Allows a first boot.
    * Used as a recovery / re-install partition (with an added GRUB manu entry).
  * `agentdata`: OCP release images payload.
* Note that sizes may change, depending on the configured `diskSizeGB` and the selected OpenShift version configured in `appliance-config.yaml` (described below).

### Factory
* This is where the disk image is written to the disk using tools such as `dd`.
* As mentioned above, the image is generic. Thus, the same image can be used on multiple servers for multiple clusters, assuming they have the same disk size.

### User site
* This is where the cluster will be deployed.
* The user boots the machine and mounts the configuration ISO (cluster configuration).
* The OpenShift installation will run until completion.

## Disk Image Build - Lab 

### Set Environment

**:warning: Use absolute directory paths.**
  ```shell
  export APPLIANCE_IMAGE="quay.io/edge-infrastructure/openshift-appliance"
  export APPLIANCE_ASSETS="/home/test/appliance_assets"
  ```

### Get `openshift-appliance` container image:
  ```shell
  podman pull $APPLIANCE_IMAGE
  ```

### Generate a template of the appliance config
A configuration file named `appliance-config.yaml` is required for running `openshift-appliance`.
  ```shell
  podman run --rm -it --pull newer -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE generate-config
  ```
Result:
```shell
INFO Generated config file in assets directory: appliance-config.yaml
```

### Set `appliance-config`
* Initially, the template will include comments about each option and will look as follows:
* Check the [appliance-config](./appliance-config.md) details on how to set each parameter.

```yaml
#
# Note: This is a sample ApplianceConfig file showing
# which fields are available to aid you in creating your
# own appliance-config.yaml file.
#
apiVersion: v1beta1
kind: ApplianceConfig
ocpRelease:
  # OCP release version in major.minor or major.minor.patch format
  # (in case of major.minor - latest patch version will be used)
  # If the specified version is not yet available, the latest supported version will be used.
  version: ocp-release-version
  # OCP release update channel: stable|fast|eus|candidate
  # Default: stable
  # [Optional]
  channel: ocp-release-channel
  # OCP release CPU architecture: x86_64|aarch64|ppc64le
  # Default: x86_64
  # [Optional]
  cpuArchitecture: cpu-architecture
# Virtual size of the appliance disk image.
# If specified, should be at least 150GiB.
# If not specified, the disk image should be resized when 
# cloning to a device (e.g. using virt-resize tool).
# [Optional]
diskSizeGB: disk-size
# PullSecret required for mirroring the OCP release payload
pullSecret: pull-secret
# Public SSH key for accessing the appliance during the bootstrap phase
# [Optional]
sshKey: ssh-key
# Password of user 'core' for connecting from console
# [Optional]
userCorePass: user-core-pass
# Local image registry details (used when building the appliance)
# [Optional]
imageRegistry:
  # The URI for the image
  # Default: docker.io/library/registry:2
  # Alternative: quay.io/libpod/registry:2.8
  # [Optional]
  uri: uri
  # The image registry container TCP port to bind. A valid port number is between 1024 and 65535.
  # Default: 5005
  # [Optional]
  port: port
# Enable all default CatalogSources (on openshift-marketplace namespace).
# Should be disabled for disconnected environments.
# Default: false
# [Optional]
enableDefaultSources: enable-default-sources
# Stop the local registry post cluster installation.
# Note that additional images and operators won't be available when stopped.
# Default: false
# [Optional]
stopLocalRegistry: stop-local-registry
# Additional images to be included in the appliance disk image.
# [Optional]
additionalImages:
   - name: image-url
# Operators to be included in the appliance disk image.
# See examples in https://github.com/openshift/oc-mirror/blob/main/docs/imageset-config-ref.yaml.
# [Optional]
operators:
  - catalog: catalog-uri
    packages:
      - name: package-name
        channels:
          - name: channel-name
```
* Modify it based on your needs. Note that:
  * `diskSizeGB`: Must be set according to the actual server disk size. If you have several server specs, you need an appliance image per each spec.
  * `ocpRelease.channel`: OCP release [update channel](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.13/html/updating_clusters/understanding-upgrade-channels-releases#understanding-upgrade-channels_understanding-upgrade-channels-releases) (stable|fast|eus|candidate)
  * `pullSecret`: May be obtained from https://console.redhat.com/openshift/install/pull-secret (requires registration).
  * `imageRegistry.uri`: Change it only if needed, otherwise the default should work.
  * `imageRegistry.port`: Change the port number in case another app uses TCP 5005. 
#### `appliance-config.yaml` Example:
```yaml
apiVersion: v1beta1
kind: ApplianceConfig
ocpRelease:
  version: 4.14
  channel: candidate
  cpuArchitecture: x86_64
diskSizeGB: 200
pullSecret: '{"auths":{<redacted>}}'
sshKey: <redacted>
userCorePass: <redacted>
```

### Add custom manifests (Optional)
* Note that any manifest added here will apply to **any** of the clusters installed using this image.
* Find more details and additional examples in OpenShift documentation: 
  * [Customizing nodes](https://docs.openshift.com/container-platform/4.13/installing/install_config/installing-customizing.html)
  * [Using MachineConfig objects to configure nodes](https://docs.openshift.com/container-platform/4.13/post_installation_configuration/machine-configuration-tasks.html#using-machineconfigs-to-change-machines)

1. Create the openshift manifests directory
```shell
mkdir $APPLIANCE_ASSETS:/openshift
```

### Include additional images (Optional)

Add any additional images that should be included as part of the appliance disk image.
These images will be pulled during the oc-mirror procedure that downloads the release images.

E.g. Use the `additionalImages` array in `appliance-config.yaml` as follows:
```shell
additionalImages:
  - name: quay.io/openshift/origin-cli
``` 

After installing the cluster, images should be available for pulling using the image digest.
E.g.
```shell
podman pull quay.io/openshift/origin-cli@sha256:b66f4289061afa26a686f77da47e2b81420959d71b21b22617a2e6a3226f6ae86ae8
```

### Include and install operators (Optional)

#### Include operators in the appliance

Operators packages can be included in the appliance disk image using the `operators` property in `appliance-config.yaml`. The relevant images will be pulled during the oc-mirror procedure, and the appropriate CatalogSources and ImageContentSourcePolicies will be automatically created in the installed cluster.

E.g. To include elasticsearch-operator from `redhat-operators` catalog:
```shell
operators:
  - catalog: registry.redhat.io/redhat/redhat-operator-index:v4.14
    packages:
      - name: elasticsearch-operator
        channels:
          - name: stable
```

#### Install operators in cluster

To automatically install the included operators during cluster installation, add the relevant custom manifests to $APPLIANCE_ASSETS:/openshift.

E.g. Cluster manifests to install OpenShift Elasticsearch Operator:

*openshift/namespace.yaml*
```shell
apiVersion: v1
kind: Namespace
metadata:
  name: operators
  labels:
    name: operators
```

*openshift/operatorgroup.yaml*
```shell
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: operator-group
  namespace: operators
spec:
  targetNamespaces:
  - operators
```

*openshift/subscription.yaml*
```shell
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: elasticsearch
  namespace: operators
spec:
  installPlanApproval: Automatic
  name: elasticsearch-operator
  source: redhat-operator-index
  channel: stable
  sourceNamespace: openshift-marketplace
```

2. Add one or more custom manifests under `$APPLIANCE_ASSETS:/openshift`.
#### MachineConfig example:
```yaml
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: master
  name: 50-master-custom-file-factory
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
        - contents:
            source: data:text/plain;charset=utf-8;base64,dGhpcyBjb250ZW50IGNhbWUgZnJvbSBidWlsZGluZyB0aGUgYXBwbGlhbmNlIGltYWdlCg==
          mode: 420
          path: /etc/custom_factory1.txt
          overwrite: true
```

### Build the disk image
* Make sure you have enough free disk space.
  * The amount of space needed is defined by the configured `diskSizeGB` value mentioned above, which is at least 150GiB.
* Building the image may take several minutes.
* The option `--privileged` is used because the `openshift-appliance` container needs to use `guestfish` to build the image.
* The option `--net=host` is used because the `openshift-appliance` container needs to use the host networking for the image registry container it runs as a part of the build process.
```shell
sudo podman run --rm -it --pull newer --privileged --net=host -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE build
```

Result
```shell
INFO Successfully downloaded CoreOS ISO
INFO Successfully generated recovery CoreOS ISO
INFO Successfully pulled container registry image
INFO Successfully pulled OpenShift 4.14.0-rc.0 release images required for bootstrap
INFO Successfully pulled OpenShift 4.14.0-rc.0 release images required for installation
INFO Successfully generated data ISO
INFO Successfully downloaded appliance base disk image
INFO Successfully extracted appliance base disk image
INFO Successfully generated appliance disk image
INFO Time elapsed: 8m0s
INFO
INFO Appliance disk image was successfully created in assets directory: assets/appliance.raw
INFO
INFO Create configuration ISO using: openshift-install agent create config-image
INFO Download openshift-install from: https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.14.0-rc.0/openshift-install-linux.tar.gz
```

### Rebuild

Before rebuilding the appliance, e.g. for changing `diskSizeGB` or `ocpRelease`, use the `clean` command. This command removes the temp folder and prepares the `assets` folder for a rebuild. 
Note: the command keeps the `cache` folder under `assets` intact.
```shell
sudo podman run --rm -it -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE clean
```

#### Demo
[![asciicast](https://asciinema.org/a/591871.svg)](https://asciinema.org/a/591871)

## Clone appliance disk image to a device (Factory)

### Baremetal servers

#### Clone disk image as-is (when 'diskSizeGB' is specified in appliance-config)
Use a tool like `dd` to clone the disk image.
E.g.
```shell
dd if=appliance.raw of=/dev/sdX bs=1M status=progress
```
This will clone the appliance disk image onto sdX. To initiate the cluster installation, boot the machine from the sdX device.

#### Resize and clone disk image (when 'diskSizeGB' is not specified in appliance-config)
Use virt-resize tool to resize and clone the disk image.
E.g.
```shell
export APPLIANCE_IMAGE="quay.io/edge-infrastructure/openshift-appliance"
export APPLIANCE_ASSETS="/home/test/appliance_assets"
export TARGET_DEVICE="/dev/sda"
sudo podman run --rm -it --privileged --net=host -v $APPLIANCE_ASSETS:/assets --entrypoint virt-resize $APPLIANCE_IMAGE --expand /dev/sda4 /assets/appliance.raw $TARGET_DEVICE --no-sparse
```
This will resize and clone the disk image onto the specified `TARGET_DEVICE`. To initiate the cluster installation, boot the machine from the `TARGET_DEVICE`.

:warning: Note: if the target device is empty (zeroed) before cloning, the `--no-sparse` flag can be removed (which will improve the cloning speed).

#### Boot from deployment ISO

As an alternative to manually cloning the disk image, see [Deployment ISO](#deployment-iso) section for instructions to generate an ISO that automates the flow.

### Virtual machines
Configure the disk to use `/path/to/appliance.raw`

## OpenShift cluster installation (User Site)

### Download `openshift-install`
* So far, the generated image has been completely generic. To install the cluster, the installer will need cluster-specific configuration.
* In order to generate the configuration image using `openshift-install`, download the binary from the URL specified in build output.
E.g. for `4.14.0-rc.0`
https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.14.0-rc.0/openshift-install-linux.tar.gz

### Generate a Cluster Configuration Image

#### Create config yamls

* Create a configuration directory

**:warning: Use absolute directory paths.**
  ```shell
  export CLUSTER_CONFIG=/home/test/cluster_config
  mkdir $CLUSTER_CONFIG && cd $CLUSTER_CONFIG
  ```
* Place both `install-config.yaml` and `agent-config.yaml` files in that directory.
* Find examples in:
  * [Appliance README](../README.md#examples)
  * [OpenShift Documentation](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/preparing-to-install-with-agent-based-installer.html#installation-bare-metal-agent-installer-config-yaml_preparing-to-install-with-agent-based-installer)
  * [Static Networking](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/preparing-to-install-with-agent-based-installer.html#static-networking)

Note: for disconnected environments, specify a dummy pull-secret in install-config.yaml (e.g. `'{"auths":{"":{"auth":"dXNlcjpwYXNz"}}}'`).

#### Add custom manifests (Optional)
* Note that any manifest added here will apply **only** to the cluster installed using this config-iso.
* Find more details and additional examples in OpenShift documentation:
  * [Customizing nodes](https://docs.openshift.com/container-platform/4.13/installing/install_config/installing-customizing.html)
  * [Using MachineConfig objects to configure nodes](https://docs.openshift.com/container-platform/4.13/post_installation_configuration/machine-configuration-tasks.html#using-machineconfigs-to-change-machines)
1. Create the openshift manifests directory
```shell
mkdir $CLUSTER_CONFIG/openshift
```

2. Add one or more custom manifests under `$CLUSTER_CONFIG/openshift`. Same as in [this MachineConfig example](user-guide.md#MachineConfig-example)

#### Generate config-image

When ready, generate the config ISO.

  :warning: The following command will delete the `install-config.yaml` and `agent-config.yaml` files - back them up first.

  ```shell
  ./openshift-install agent create config-image --dir $CLUSTER_CONFIG
  ```

The content of `cluster_config` directory should be
  ```shell
  ├── agentconfig.noarch.iso
  ├── auth
  │   ├── kubeadmin-password
  │   └── kubeconfig
  ```

Note: The config ISO contains configurations and cannot be used as a bootable ISO.

### Mount
**:warning: Ensure nodes have sufficient vCPUs and memory, see [requirements](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/preparing-to-install-with-agent-based-installer.html#recommended-resources-for-topologies).**
* Mount the `agentconfig.noarch.iso` as a CD-ROM on every node, or attach it using a USB stick.
* Start the machine(s)

### Demo
[![asciicast](https://asciinema.org/a/590824.svg)](https://asciinema.org/a/590824)

### Monitor installation
Use `openshift-install` to monitor the bootstrap and installation process

#### Monitor the bootstrap process
  ```shell
  ./openshift-install --dir $CLUSTER_CONFIG agent wait-for bootstrap-complete
  ```
* Review OpenShift documentation: [Waiting for the bootstrap process to complete](https://docs.openshift.com/container-platform/4.13/installing/installing_platform_agnostic/installing-platform-agnostic.html#installation-installing-bare-metal_installing-platform-agnostic)
#### Monitor the installation process
  ```shell
  ./openshift-install --dir $CLUSTER_CONFIG agent wait-for install-complete
  ```
* Review OpenShift documentation: [Completing installation on user-provisioned infrastructure](https://docs.openshift.com/container-platform/4.13/installing/installing_platform_agnostic/installing-platform-agnostic.html#installation-complete-user-infra_installing-platform-agnostic)

### Access Cluster

**:warning: Ensure the server domain in $CLUSTER_CONFIG/auth/kubeconfig is resolvable.**

```shell
export KUBECONFIG=$CLUSTER_CONFIG/auth/kubeconfig
```

#### Confirm that the cluster version is available:
```shell
oc get clusterverison
```

#### Confirm that all cluster components are available:
``` shell
oc get clusteroperator
```

### Recovery / Reinstall
* To reinstall the cluster using the above-mentioned `agentboot` partition, reboot all the nodes and select the `Recovery: Agent-Based Installer` option.
![grub.png](images%2Fgrub.png)

## Deployment ISO

To simplify the deployment process of the appliance disk image (appliance.raw), the deployment ISO can be used. Upon booting a machine with this ISO, the appliance disk image would be automatically cloned into the specified target device.

To build the ISO, appliance.raw disk image should be available under `assets` directory. I.e. the appliance disk image should be first built.

:warning: Note: the appliance.raw should be built without specifying `diskSizeGB` property in appliance-config.yaml

### Build

Use the 'build iso' command for generating the ISO:
```shell
export APPLIANCE_IMAGE="quay.io/edge-infrastructure/openshift-appliance"
export APPLIANCE_ASSETS="/home/test/appliance_assets"
sudo podman run --rm -it --privileged -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE build iso --target-device /dev/sda
```

The result should be an appliance.iso file under `assets` directory. To initiate the deployment, attach/mount the ISO to the machine and boot it. After the deployment is completed, boot from the target device to start cluster installation.


The command supports the following flags:
```
--target-device string    Target device name to clone the appliance into (default "/dev/sda")
--post-script string      Script file to invoke on completion (should be under assets directory)
--sparse-clone            Use sparse cloning - requires an empty (zeroed) device
--dry-run                 Skip appliance cloning (useful for getting the target device name)
```

### Examples

#### --post-script

To perform post deployment operations create a bash script file under assets directory.

E.g. shutting down the machine post deployment

```bash
cat $APPLIANCE_ASSETS/post.sh

#!/bin/bash
echo Shutting down the machine...
shutdown -a
```

```bash
sudo podman run --rm -it --privileged -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE build iso --post-script post.sh
```

### Demo

![deploy-iso.gif](images/deploy-iso.gif)
