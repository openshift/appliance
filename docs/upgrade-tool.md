# upgrade-tool
**WARNING: This utility is currently in experimental status and is not supported**.

The `upgrade-tool` utility provides a method to upgrade an Openshift cluster without a registry. This can be very useful in offline or limited environments.

In the first stage we use the tool to prepare a bundle with all the images and information necessary for the upgrade. This bundle will be dumped to an external device (a USB stick for example), and physically connected to the Openshift node as update source.

In a second stage, once the upgrade bundle is available, we can start the upgrade process on the Openshift cluster.
## Installation
The source code is available n the [upgrade-tool repository](https://github.com/jhernand/upgrade-tool).  
Additionally we have a container ready to use at [quay.io/jhernand/upgrade-tool](https://quay.io/repository/jhernand/upgrade-tool).

## Create the bundle
The `upgrade-tool` has a CLI utility to create a bundle file with all the information and images needed to upgrade an Openshift cluster to a specifc version.

To create this bundle we will need the following data:
* **architecture**. OCP release CPU architecture: `x86_64`, `aarch64`, `ppc64le`.
* **version**. OCP release version to which we want to upgrade our cluster.
* **pull-secret file**. File with our pull secrets to download images from the registries.

With this data we can execute the command to generate the bundle:
```shell
upgrade-tool create bundle --arch=x86_64 --version=4.14.12 --pull-secret=pull-secret.json --output .
```

The tool will create a bundle, a checksum file, and also a manifest with the Openshift resources needed to start the cluster upgrade in the next step in the current directory.
```shell
.
├── upgrade-4.14.12-x86_64.sha256
├── upgrade-4.14.12-x86_64.tar
└── upgrade-4.14.12-x86_64.yaml
```

## Prepare the upgrade device
The `upgrade-tool` controller, deployed on the Openshift cluster, expects to extract the upgrade data from a device with the tar file. Then we need to copy this data to a device, we can use tools like `dd` or `cp` to do it.
For example:
```shell
dd if=upgrade-4.14.12-x86_64.tar of=/dev/sda bs=4M status=progress
```

## Upgrade the cluster
Once the upgrade device is ready, we connect it to the node(s) we want upgrade.
We can check thename assigned to the device using `dmesg`:
```
# dmesg

 ... ...

[262375.179857] sd 1:0:0:0: Attached scsi generic sg0 type 0
[262375.180053] sd 1:0:0:0: Power-on or device reset occurred
[262375.180315] sd 1:0:0:0: [sda] 38671980 512-byte logical blocks: (19.8 GB/18.4 GiB)
[262375.180443] sd 1:0:0:0: [sda] Write Protect is off
[262375.180444] sd 1:0:0:0: [sda] Mode Sense: 63 00 00 08
[262375.180577] sd 1:0:0:0: [sda] Write cache: enabled, read cache: enabled, doesn't support DPO or FUA
[262375.182288] sd 1:0:0:0: [sda] Attached SCSI disk
```

And using `tar` we should be able to see the files inside the bundle:
```
# tar tvf /dev/sda
-rw-r--r-- root/root     23043 2024-02-16 16:05 metadata.json
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_uploads/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/904d74140e9e6872cebbc19000dae5071ba9174161cbee64bf3d97d0cfd8fe6b/
-rw-r--r-- root/root        71 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/904d74140e9e6872cebbc19000dae5071ba9174161cbee64bf3d97d0cfd8fe6b/link
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/d8190195889efb5333eeec18af9b6c82313edd4db62989bd3a357caca4f13f0e/
-rw-r--r-- root/root        71 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/d8190195889efb5333eeec18af9b6c82313edd4db62989bd3a357caca4f13f0e/link
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/406c47e5677b63b166cdf8a4e7a4980eeab4566405ea7b11a3000f16487a08e6/
-rw-r--r-- root/root        71 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/406c47e5677b63b166cdf8a4e7a4980eeab4566405ea7b11a3000f16487a08e6/link
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/793a6602df09d9797b2bb595d5d98c60d46e83df7dba4c96866373e08723126e/
-rw-r--r-- root/root        71 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/793a6602df09d9797b2bb595d5d98c60d46e83df7dba4c96866373e08723126e/link
drwxr-xr-x root/root         0 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/9b66199101ad56591703149dad8a0de6cb78d17d94ccb0c50f8e3f66eace0b51/
-rw-r--r-- root/root        71 2024-02-16 16:00 docker/registry/v2/repositories/openshift-release-dev/ocp-release/_layers/sha256/9b66199101ad56591703149dad8a0de6cb78d17d94ccb0c50f8e3f66eace0b51/link

... ...
```

Now that we know the name and we have verified the device, we install the `upgrade-tool` controller on the cluster using the manifest file created by the tool.
```shell
oc apply -f upgrade-4.14.12-x86_64.yaml
```

If everything has worked correctly we can see a `controller` pod in the `upgrade-tool` namespace:
```shell
NAME                                      READY   STATUS      RESTARTS   AGE
controller                                1/1     Running     0          11m
```

The next step would be to start the extractor and loader using the controller. Nowadays it is triggered using an annotation on the `clusterversion` object. In the annotation we will configure the device name with bundle data:
```shell
oc patch --type=merge --patch='{"metadata":{"annotations":{"upgrade-tool/bundle-file":"/dev/sda"}}}' clusterversion/version
```

The controller will start the next processes to prepare the cluster upgrade:
1. A pod named **bundle-extractor** will extract the bundle data to a local directory.
2. A pod with a embedded registry **server**, will be started to make images available.
3. The next pod, called **bundle-loader**, will copy the images from the embedded registry to the CRI-O cache.
```shell
NAME                                      READY   STATUS      RESTARTS   AGE
bundle-extractor-appliance-node-1-7cr7q   0/1     Completed   0          14m
bundle-server-5vr2p                       0/1     Completed   0          14m
bundle-loader-appliance-node-1-bhr4q      0/1     Completed   1          13m
controller                                1/1     Running     0          14m
```

We can track the progress using the annotations values on the cluster nodes:
```shell
watch -n 5 oc get nodes -o custom-columns=NAME:.metadata.name,EXTRACTED:.metadata.labels.upgrade-tool/bundle-extracted,LOADED:.metadata.labels.upgrade-tool/bundle-loaded,PROGRESS:.metadata.annotations.upgrade-tool/progress
```
Output:
```
NAME               EXTRACTED   LOADED   PROGRESS
appliance-node-1   true        true     Pulled 189 of 189 images
```

When these processes are completed, the `controller` will update the `spec` of `clusterversion` object forcing the cluster to upgrade to the new version.
