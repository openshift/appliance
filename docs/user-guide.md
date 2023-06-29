# OpenShift Appliance User Guide

## What is OpenShift Appliance
The `openshift-appliance` is a container-based utility that builds a disk image that facilitates the installation of OpenShift with an [Agent-based installer](https://cloud.redhat.com/blog/meet-the-new-agent-based-openshift-installer-1).

Its main purpose is to enable a completely offline installation of an OpenShift cluster, Thus, all required images are included in the appliance disk image.


## Download
OpenShift Appliance is available for download at: https://quay.io/edge-infrastructure/openshift-appliance

## High-Level Flow Overview

![hl-overview.png](images%2Fhl-overview.png)

### Lab
* This is where `openshift-appliance` gets used to create a raw space disk image.
  * **raw:** so it can be copied as-is to multiple servers. 
  * **sparse:** to keep the size as small as possible.
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
    * Used as a recovery / re-install partition (with an added grub manu entry).
  * `agentdata`: OCP release images payload.
* Note that sizes may change, depending on the configured `diskSizeGB` and the selected OpenShift version configured in `appliance-config.yaml`
* It’s important to note that the sizes may vary based on the `diskSizeGB` configuration and the OpenShift version selected in the `appliance-config.yaml` file (detailed later on).

### Factory
* This is where the disk image is written to the disk using tools such as `dd`.
* As mentioned above, the created image is generic. Thus, the same image can be used on multiple servers, assuming they have the same disk size.

### User site
* This is where the user boots the machine and mounts the configuration ISO (cluster configuration).
* The OpenShift installation will run until completion.

:warning: note that the openshift-appliance disk image supports UEFI boot mode only.


## Disk Image Build - Lab 

### Set Environment

**:warning: use absolute paths.**
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
  podman run --rm -it -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE generate-config
  ```
Result:
```shell
INFO Generated config file in assets directory: appliance-config.yaml
```
  

### Set `appliance-config`
* Initially, the template will include comments about each option and will look as follows:

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
  version: ocp-release-version
  # OCP release update channel: stable|fast|eus|candidate
  # Default: stable
  # [Optional]
  channel: ocp-release-channel
  # OCP release CPU architecture: x86_64|aarch64|ppc64le
  # Default: x86_64
  # [Optional]
  cpuArchitecture: cpu-architecture
# Virtual size of the appliance disk image (at least 150GiB)
diskSizeGB: disk-size
# PullSecret required for mirroring the OCP release payload
pullSecret: pull-secret
# Public SSH key for accessing the appliance
# [Optional]
sshKey: ssh-key
# Password of user 'core' for connecting from console
# [Optional]
userCorePass: user-core-pass
```
* Modify it based on your needs. Note that:
  * The preferred version is 4.12.z until the `openshift-install` bug [OCPBUGS-14900](https://issues.redhat.com/browse/OCPBUGS-14900 ) gets resolved.
  * `diskSizeGB`: Must be set according to the actual server disk size. If you have several server specs, you need an appliance image per each spec.
  * `ocpRelease.channel`: OCP release [update channel](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.13/html/updating_clusters/understanding-upgrade-channels-releases#understanding-upgrade-channels_understanding-upgrade-channels-releases) (stable|fast|eus|candidate)
  * `pullSecret`: May be obtained from https://console.redhat.com/openshift/install/pull-secret (requires registration).

#### `appliance-config.yaml` Example:
```yaml
apiVersion: v1beta1
kind: ApplianceConfig
ocpRelease:
  version: 4.12.10
  channel: stable
  cpuArchitecture: x86_64
diskSizeGB: 200
pullSecret: <redacted>
sshKey: <redacted>
userCorePass: <redacted>
```

### Build the image
* Make sure you have enough free disk space.
* Building the image may take several minutes.
* The option `--privileged` is used because the `openshift-appliance` container needs to use `guestfish` to build the image.
* The option `--net=host` is used because the `openshift-appliance` container needs to use the host networking for the image registry container it runs as a part of the build process.
```shell
podman run --rm -it --privileged --net=host -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE build

```
Result
```shell
INFO Successfully downloaded CoreOS ISO
INFO Successfully generated recovery CoreOS ISO
INFO Successfully pulled container registry image
INFO Successfully pulled OpenShift 4.12.10 release images required for bootstrap
INFO Successfully pulled OpenShift 4.12.10 release images required for installation
INFO Successfully generated data ISO
INFO Successfully downloaded appliance base disk image
INFO Successfully extracted appliance base disk image
INFO Successfully generated appliance disk image
INFO Time elapsed: 8m0s
INFO
INFO Appliance successfully created in assets directory: assets/appliance.raw
INFO
INFO Create configuration ISO using: openshift-install agent create config-image
INFO Download openshift-install from: https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.12.10/openshift-install-linux.tar.gz
```

#### Demo
[![asciicast](https://asciinema.org/a/591871.svg)](https://asciinema.org/a/591871)

## Copy Appliance Image to Disk - Factory
* Baremetal servers: use `dd`
* Virtual Machines: configure the disk to use `/path/to/appliance.raw`

## OpenShift Cluster Install - User Site

### Download `openshift-install`
* So far, the generated image has been completely generic. To install the cluster, the installer will need cluster-specific configuration.
* The `openshift-install` version mentioned above does not include the config-image API. This API is available starting from version `4.14`.
* * Download and extract the latest `openshift-install` version from: https://mirror.openshift.com/pub/openshift-v4/clients/ocp-dev-preview/latest/openshift-install-linux.tar.gz
```shell
  curl -O https://mirror.openshift.com/pub/openshift-v4/clients/ocp-dev-preview/latest/openshift-install-linux.tar.gz
  tar -xvf openshift-install-linux.tar.gz
  ```

### Generate a Cluster Configuration Image
* Create a configuration directory

**:warning: use absolute paths.**
  ```shell
  export CLUSTER_CONFIG=/home/test/cluster_config
  mkdir $CLUSTER_CONFIG && cd $CLUSTER_CONFIG
  ```
* Place both `install-config.yaml` and `agent-config.yaml` files in that directory.
* Find examples in:
  * [Appliance README](../README.md#examples)
  * [OpenShift Documentation](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/preparing-to-install-with-agent-based-installer.html#installation-bare-metal-agent-installer-config-yaml_preparing-to-install-with-agent-based-installer)
  * [Static Networking](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_ba[…]taller/preparing-to-install-with-agent-based-installer.html), currently blocked by [OCPBUGS-15637](https://issues.redhat.com/browse/OCPBUGS-15637)
* When ready, generate the config-iso.
  ```shell
  ./openshift-install agent create config-image --dir $CLUSTER_CONFIG
  ```
* Note: The config ISO contains configurations and cannot be used as a bootable ISO.

The content of `cluster_config` directory should be
  ```shell
  ├── agentconfig.noarch.iso
  ├── auth
  │   ├── kubeadmin-password
  │   └── kubeconfig
  ```

### Mount
* Mount the `agentconfig.noarch.iso` as a CD-ROM on every node, or attach it using a USB stick.
* Start the machine(s)

### Demo
[![asciicast](https://asciinema.org/a/590824.svg)](https://asciinema.org/a/590824)

### Monitor installation
#### Use `openshift-install` to monitor the bootstrap process
  ```shell
  export KUBECONFIG=$CLUSTER_CONFIG/auth/kubeconfig 
  ./openshift-install --dir $CLUSTER_CONFIG  wait-for bootstrap-complete
  ```
* Review OpenShift documentation: [Waiting for the bootstrap process to complete](https://docs.openshift.com/container-platform/4.13/installing/installing_platform_agnostic/installing-platform-agnostic.html#installation-installing-bare-metal_installing-platform-agnostic)
#### Monitor the installation process
  ```shell
  ./openshift-install --dir $CLUSTER_CONFIG wait-for install-complete
  ```
* Review OpenShift documentation: [Completing installation on user-provisioned infrastructure](https://docs.openshift.com/container-platform/4.13/installing/installing_platform_agnostic/installing-platform-agnostic.html#installation-complete-user-infra_installing-platform-agnostic)

### Recovery / Reinstall
* To reinstall the cluster using the above-mentioned `agentboot` partition, reboot all the nodes and select the `Recovery: Agent-Based Installer` option.
![grub.png](images%2Fgrub.png)
