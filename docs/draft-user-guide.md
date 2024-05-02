The `openshift-appliance` utility enables self-contained OpenShift Container Platform cluster installations, meaning that it does not rely on internet connectivity or external registries. It is a container-based utility that builds a disk image that includes the [Agent-based Installer](https://cloud.redhat.com/blog/meet-the-new-agent-based-openshift-installer-1). The disk image can then be used to install multiple OpenShift Container Platform clusters.

## Downloading the OpenShift-based Appliance builder
The OpenShift-based Appliance builder is available for download at: [https://catalog.redhat.com/software/containers/assisted/agent-preinstall-image-builder-rhel9/65a55174031d94dbea7f2e00](https://catalog.redhat.com/software/containers/assisted/agent-preinstall-image-builder-rhel9/65a55174031d94dbea7f2e00)

## High level overview

[image=[src="images/hl-overview.png", alt="A high level overview of the OpenShift-based Appliance Builder", size="LG - Large", data-cp-size="100%", caption="A high level overview of the OpenShift-based Appliance Builder", ]]

### Lab
This is where `openshift-appliance` is used to create a raw sparse disk image.

  * **Raw**: So it can be copied as-is to multiple servers.
  * **Sparse**: To minimize the physical size of the image.

The end result is a generic disk image with a partition layout as follows:
```
Name       Type        VFS      Label       Size  Parent
/dev/sda2  filesystem  vfat     EFI-SYSTEM  127M  -
/dev/sda3  filesystem  ext4     boot        350M  -
/dev/sda4  filesystem  xfs      root        180G  -
/dev/sda5  filesystem  ext4     agentboot   1.2G  -
/dev/sda6  filesystem  iso9660  agentdata   18G   -
```

The two additional partitions:

  * `agentboot`: The Agent-based Installer ISO.
    * Allows a first boot.
    * Used as a recovery/re-install partition (with an added GRUB menu entry).
  * `agentdata`:The OCP release images payload.

Note that sizes may change, depending on the configured `diskSizeGB` and the selected OpenShift version configured in `appliance-config.yaml` (described below).

### Factory
This is where the disk image is written to the disk using tools such as `dd`. Since the image is generic, the same image can be used on multiple servers for multiple clusters, assuming they have the same disk size.

### User site
This is where the cluster will be deployed. The user boots the machine and mounts the configuration ISO (cluster configuration). The OpenShift installation will run until completion.

## Building the Disk Image - Lab

### Creating and configuring manifest files

1. Set the Environment by running the following commands.
  **Warning**: Use absolute directory paths.

  ```shell
  $ export APPLIANCE_IMAGE="catalog.redhat.com/software/containers/assisted/agent-preinstall-image-builder-rhel9/65a55174031d94dbea7f2e00"
  $ export APPLIANCE_ASSETS="/home/test/appliance_assets"
  ```

2. Get the `openshift-appliance` container image by running the following command:
  ```shell
  $ podman pull $APPLIANCE_IMAGE
  ```

3. Generate a template of the appliance config by running the following command:
  ```shell
  $ podman run --rm -it --pull newer -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE generate-config
  ```
  Result:
  ```shell
  INFO Generated config file in assets directory: appliance-config.yaml
  ```
  This generates the `appliance-config.yaml` file required for running `openshift-appliance`.

4. Configure the `appliance-config.yaml` file. Initially, the template will include comments about each option and will look like the following example:

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
  * Modify the file based on your needs, where:
      * `diskSizeGB`: Specifies the actual server disk size. If you have several server specs, you need an appliance image per each spec.
      * `ocpRelease.channel`: Specifies the OCP release [update channel](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.15/html/updating_clusters/understanding-openshift-updates-1#understanding-update-channels-releases) (stable|fast|eus|candidate)
      * `pullSecret`: Specifies the pull secret. This may be obtained from https://console.redhat.com/openshift/install/pull-secret (requires registration).
      * `imageRegistry.uri`: Specifies the URI for the image. Change it only if needed, otherwise the default should work.
      * `imageRegistry.port`: Specifies the image registry container TCP port to bind. Change the port number in case another app uses TCP 5005.

    `appliance-config.yaml` Example:
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
    **Note**: Currently only `x86_64` CPU architecture is supported.

### Optional: Add custom manifests

Any manifest added here will apply to **all** of the clusters installed using this image.
**Note**: Find more details and additional examples in OpenShift Container Platform documentation:
* [Customizing nodes](https://docs.openshift.com/container-platform/4.13/installing/install_config/installing-customizing.html)
* [Using MachineConfig objects to configure nodes](https://docs.openshift.com/container-platform/4.13/post_installation_configuration/machine-configuration-tasks.html#using-machineconfigs-to-change-machines)

1. Create the openshift manifests directory by running the following command:

  ```shell
  $ mkdir ${APPLIANCE_ASSETS}/openshift
  ```

2. Add one or more custom manifests under `${APPLIANCE_ASSETS}/openshift`.

  MachineConfig example:
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

### Optional: Including additional images

Add any additional images that should be included as part of the appliance disk image. These images will be pulled during the oc-mirror procedure that downloads the release images.

The following example includes images for the Apache HTTP server daemon and the OpenShift CLI.

1. Specify images in the `additionalImages` array in `appliance-config.yaml`:
  ```shell
  additionalImages:
    - name: quay.io/fedora/httpd-24
    - name: quay.io/openshift/origin-cli
  ```
  After installing the cluster, images should be available for pulling using the image digest.

2. Fetch the image digests by using skopeo from inside the node:
  ```shell
  $ skopeo inspect docker://registry.appliance.com:5000/fedora/httpd-24 | jq .Digest
  "sha256:5d98ffbb97ea86633aed7ae2445b9d939e29639a292d3052efb078e72606ba04"
  ```

3. Pull the images by running the following command:
  ```shell
  $ podman pull quay.io/fedora/httpd-24@sha256:5d98ffbb97ea86633aed7ae2445b9d939e29639a292d3052efb078e72606ba04
  ```

After installation, the image can be used, for example, to create a new application:
```shell
$ oc --kubeconfig auth/kubeconfig new-app --name httpd --image quay.io/fedora/httpd-24@sha256:5d98ffbb97ea86633aed7ae2445b9d939e29639a292d3052efb078e72606ba04 --allow-missing-images
```
```shell
$ oc --kubeconfig auth/kubeconfig get deployment
NAME    READY   UP-TO-DATE   AVAILABLE
httpd   1/1     1            1
```

### Optional: Including and installing Operators

#### Including Operators in the appliance

Operators packages can be included in the appliance disk image using the `operators` property in `appliance-config.yaml`. The relevant images will be pulled during the oc-mirror procedure, and the appropriate CatalogSources and ImageContentSourcePolicies will be automatically created in the installed cluster.

For example, to include the `elasticsearch-operator` from `redhat-operators` catalog, add the following entry to the `appliance-config.yaml` file:
```yaml
operators:
  - catalog: registry.redhat.io/redhat/redhat-operator-index:v4.14
    packages:
      - name: elasticsearch-operator
        channels:
          - name: stable-5.8
```

For each Operator, ensure the name and channel are correct by listing the available Operators in catalog:
```bash
$ oc-mirror list operators --catalog=registry.redhat.io/redhat/redhat-operator-index:v4.14
```

#### Installing Operators in the cluster

To automatically install the included operators during cluster installation, add the relevant custom manifests to `${APPLIANCE_ASSETS}/openshift`.

**Note**: These manifests will deploy the Operators for any cluster installation, meaning that the manifests will be incorporated in the appliance disk image.

For example, the following cluster manifests are used to install the OpenShift Elasticsearch Operator:

*openshift/namespace.yaml*
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: operators
  labels:
    name: operators
```

*openshift/operatorgroup.yaml*
```yaml
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
```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: elasticsearch
  namespace: operators
spec:
  installPlanApproval: Automatic
  name: elasticsearch-operator
  source: cs-redhat-operator-index
  channel: stable
  sourceNamespace: openshift-marketplace
```

**Note**: Each file should contain a single object.

### Building the disk image

Before building the disk image, consider the following information:
* Make sure you have enough free disk space.
  * The amount of space needed is defined by the configured `diskSizeGB` value, which is at least 150GiB.
* Building the image may take several minutes.
* The `--privileged` option is used because the `openshift-appliance` container needs to use `guestfish` to build the image.
* The `--net=host` option is used because the `openshift-appliance` container needs to use the host networking for the image registry container it runs as a part of the build process.

Build the disk image by running the following command:
```shell
$ sudo podman run --rm -it --pull newer --privileged --net=host -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE build
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

### Rebuilding the disk image

You can rebuild the appliance image, for example, to change `diskSizeGB` or `ocpRelease`.

Before rebuilding, use the `clean` command to remove the `temp` folder and prepare the `assets` folder for a rebuild:
```shell
$ sudo podman run --rm -it -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE clean
```

**Note**: The `clean` command keeps the `cache` folder under `assets` intact. To clean the entire cache as well, use the `--cache` flag with the `clean` command.

## Cloning the appliance disk image to a device - Factory

### Cloning the disk image for bare metal servers

#### Cloning the disk image as-is

When 'diskSizeGB' is specified in the `appliance-config.yaml` file, you can use a tool like `dd` to clone the disk image.

Example `dd` command:
```shell
$ dd if=appliance.raw of=/dev/sdX bs=1M status=progress
```
This will clone the appliance disk image onto sdX. To initiate the cluster installation, boot the machine from the sdX device.

#### Resizing and cloning the disk image

If 'diskSizeGB' is not specified in the `appliance-config.yaml` file, use virt-resize tool to resize and clone the disk image.

Example commands:
```shell
$ export APPLIANCE_IMAGE="catalog.redhat.com/software/containers/assisted/agent-preinstall-image-builder-rhel9/65a55174031d94dbea7f2e00"
$ export APPLIANCE_ASSETS="/home/test/appliance_assets"
$ export TARGET_DEVICE="/dev/sda"
$ sudo podman run --rm -it --privileged --net=host -v $APPLIANCE_ASSETS:/assets --entrypoint virt-resize $APPLIANCE_IMAGE --expand /dev/sda4 /assets/appliance.raw $TARGET_DEVICE --no-sparse
```
This will resize and clone the disk image onto the specified `TARGET_DEVICE`. To initiate the cluster installation, boot the machine from the `TARGET_DEVICE`.

**Note**: If the target device is empty (zeroed) before cloning, the `--no-sparse` flag can be removed, which will improve the cloning speed.

#### Booting from a deployment ISO

As an alternative to manually cloning the disk image, see the [Deployment ISO](#deployment-iso) section for instructions to generate an ISO that automates the flow.

### Cloning the disk image for virtual machines
To clone the disk image onto a virtual machine, configure the disk to use `/path/to/appliance.raw`.

## Installing the OpenShift Container Platform cluster - User Site

### Downloading the `openshift-install` binary
The image generated onto devices is completely generic. To install the cluster, the installer will need cluster-specific configuration.

In order to generate the configuration image using the `openshift-install` command, first download the binary from the URL specified in the build output.
For example, the URL for `4.14.0-rc.0` is: [https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.14.0-rc.0/openshift-install-linux.tar.gz](https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.14.0-rc.0/openshift-install-linux.tar.gz)

### Generating a Cluster Configuration Image

#### Creating the configuration YAML files

1. Create a configuration directory by running the following commands.

  **Warning**: Use absolute directory paths.

  ```shell
  $ export CLUSTER_CONFIG=/home/test/cluster_config
  $ mkdir $CLUSTER_CONFIG && cd $CLUSTER_CONFIG
  ```
2. Configure and place both `install-config.yaml` and `agent-config.yaml` files in the directory you created.
  You can find examples of these configuration files in the following documentation:
  * [Preparing to install with the Agent-based installer](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/preparing-to-install-with-agent-based-installer.html#installation-bare-metal-agent-installer-config-yaml_preparing-to-install-with-agent-based-installer)
    * [Static Networking](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/preparing-to-install-with-agent-based-installer.html#static-networking)
  * [Installing an OpenShift Container Platform cluster with the Agent-based Installer](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/installing-with-agent-based-installer.html)
  * [Example manifest files](#example-manifest-files)

**Note**:

* For disconnected environments, specify a dummy pull-secret in `install-config.yaml` file, for example `'{"auths":{"":{"auth":"dXNlcjpwYXNz"}}}'`.
* The SSH public key for the `core` user can be specified in the `install-config.yaml` file under the `sshKey` property. It can be used for logging in to the machines post cluster installation.

#### Optional: Adding custom manifests

1. Create the `openshift` manifests directory by running the following command:
  ```shell
  $ mkdir $CLUSTER_CONFIG/openshift
  ```

2. Add one or more custom manifests under `$CLUSTER_CONFIG/openshift`.

  Find more details and additional examples in OpenShift documentation:
  * [Customizing nodes](https://docs.openshift.com/container-platform/4.13/installing/install_config/installing-customizing.html)
  * [Using MachineConfig objects to configure nodes](https://docs.openshift.com/container-platform/4.13/post_installation_configuration/machine-configuration-tasks.html#using-machineconfigs-to-change-machines)
  * [Using ZTP manifests](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/installing-with-agent-based-installer.html#installing-ocp-agent-ztp_installing-with-agent-based-installer)

  **Note**: Any manifest added here will apply **only** to the cluster installed using this config-iso.

#### Optional: Installing Operators in the cluster

To automatically install Operators during cluster installation, add the relevant custom manifests to `$CLUSTER_CONFIG/openshift`.

**Note**: For disconnected environments, the Operators should be [included](#installing-operators-in-the-cluster) in the appliance image.

#### Generating the configuration image

When ready, generate the config ISO.

**Warning**: Creating the config image will delete the `install-config.yaml` and `agent-config.yaml` files. Make sure to back them up first.

* Generate the config ISO by running the following command:

  ```shell
  $ ./openshift-install agent create config-image --dir $CLUSTER_CONFIG
  ```

  The content of `cluster_config` directory should look like the following example:

  ```shell
  ├── agentconfig.noarch.iso
  ├── auth
  │   ├── kubeadmin-password
  │   └── kubeconfig
  ```

  **Note**: The config ISO contains only configurations and cannot be used as a bootable ISO.

### Mounting the configuration image

**Warning**: Ensure nodes have sufficient vCPUs and memory, see [requirements](https://docs.openshift.com/container-platform/4.13/installing/installing_with_agent_based_installer/preparing-to-install-with-agent-based-installer.html#recommended-resources-for-topologies).

1. Mount the `agentconfig.noarch.iso` as a CD-ROM on every node, or attach it using a USB stick.

2. Start the machine(s)

### Monitoring the installation

You can use the `openshift-install` command to monitor the bootstrap and installation process.

#### Monitoring the bootstrap process

* Monitor the bootstrap process by running the following command:

  ```shell
  $ ./openshift-install --dir $CLUSTER_CONFIG agent wait-for bootstrap-complete
  ```

  For more information, see [Waiting for the bootstrap process to complete](https://docs.openshift.com/container-platform/4.13/installing/installing_platform_agnostic/installing-platform-agnostic.html#installation-installing-bare-metal_installing-platform-agnostic).

#### Monitoring the installation process

* Monitor the installation process by running the following command:

  ```shell
  $ ./openshift-install --dir $CLUSTER_CONFIG agent wait-for install-complete
  ```
  For more information, see [Completing installation on user-provisioned infrastructure](https://docs.openshift.com/container-platform/4.13/installing/installing_platform_agnostic/installing-platform-agnostic.html#installation-complete-user-infra_installing-platform-agnostic).

### Accessing the installed cluster

You can log in to your cluster as a default system user by exporting the cluster kubeconfig file.

**Warning:** Ensure the server domain in $CLUSTER_CONFIG/auth/kubeconfig is resolvable.

* Export the `kubeadmin` credentials by running the following command:

  ```shell
  $ export KUBECONFIG=$CLUSTER_CONFIG/auth/kubeconfig
  ```

#### Confirming that the cluster version is available:

* Confirm that the cluster version is available by running the following command:

  ```shell
  $ oc get clusterverison
  ```

#### Confirming that all cluster components are available:

* Confirm that cluster components are available by running the following command:

  ``` shell
  $ oc get clusteroperator
  ```

### Recovering or reinstalling the cluster

* Reinstall the cluster using the above-mentioned `agentboot` partition by rebooting all the nodes and selecting the `Recovery: Agent-Based Installer` option:
[image=[src="images/grub_1.png", alt="User interface for recovering or reinstalling a cluster", size="LG - Large", data-cp-size="100%",  ]]


## Using a deployment ISO

To simplify the deployment process of the appliance disk image (`appliance.raw`), you can use the deployment ISO. Upon booting a machine with this ISO, the appliance disk image is automatically cloned into the specified target device.

To build the ISO, appliance.raw disk image should be available under the `assets` directory, meaning that the appliance disk image should be built first.

**Warning**: The `appliance.raw` image should be built without specifying `diskSizeGB` property in the `appliance-config.yaml` file.

### Building the ISO

1. Generate the ISO by running the following commands:

  ```shell
  $ export APPLIANCE_IMAGE="catalog.redhat.com/software/containers/assisted/agent-preinstall-image-builder-rhel9/65a55174031d94dbea7f2e00"
  $ export APPLIANCE_ASSETS="/home/test/appliance_assets"
  $ sudo podman run --rm -it --privileged -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE build iso --target-device /dev/sda
  ```

  The command supports the following flags:
  ```
  -target-device string    Target device name to clone the appliance into (default "/dev/sda")
  --post-script string      Script file to invoke on completion (should be under assets directory)
  --sparse-clone            Use sparse cloning - requires an empty (zeroed) device
  --dry-run                 Skip appliance cloning (useful for getting the target device name)
  ```
2. Verify that there is an `appliance.iso` file under the `assets` directory.

To initiate the deployment, attach or mount the ISO to the machine and boot it. After the deployment is completed, boot from the target device to start cluster installation.

### Example operations

#### Using --post-script

To perform post-deployment operations, create a bash script file under assets directory.

For example, shutting down the machine post deployment:

```bash
$ cat $APPLIANCE_ASSETS/post.sh

#!/bin/bash
echo Shutting down the machine...
shutdown -a
```

```bash
$ sudo podman run --rm -it --privileged -v $APPLIANCE_ASSETS:/assets:Z $APPLIANCE_IMAGE build iso --post-script post.sh
```

### Example manifest files

#### Creating an SNO cluster

##### agent-config.yaml

```yaml
apiVersion: v1alpha1
kind: AgentConfig
rendezvousIP: 192.168.122.100
```

##### install-config.yaml
```yaml
apiVersion: v1
metadata:
  name: appliance
baseDomain: appliance.com
controlPlane:
  name: master
  replicas: 1
compute:
- name: worker
  replicas: 0
networking:
  networkType: OVNKubernetes
  machineNetwork:
  - cidr: 192.168.122.0/24
platform:
  none: {}
pullSecret: '{"auths":{"":{"auth":"dXNlcjpwYXNz"}}}'
sshKey: 'ssh-rsa ...'
```

#### Creating a multi-node cluster

##### agent-config.yaml

```yaml
apiVersion: v1alpha1
kind: AgentConfig
rendezvousIP: 192.168.122.100
```

##### install-config.yaml
```yaml
apiVersion: v1
metadata:
  name: appliance
baseDomain: appliance.com
controlPlane:
  name: master
  replicas: 3
compute:
- name: worker
  replicas: 2
networking:
  networkType: OVNKubernetes
  machineNetwork:
  - cidr: 192.168.122.0/24
platform:
  baremetal:
    apiVIPs:
    - 192.168.122.200
    ingressVIPs:
    - 192.168.122.201
pullSecret: '{"auths":{"":{"auth":"dXNlcjpwYXNz"}}}'
sshKey: 'ssh-rsa ...'
```