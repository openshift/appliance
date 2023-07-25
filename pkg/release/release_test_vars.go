package release

var fakeCincinnatiMetadata = `

{
  "image": "quay.io/openshift-release-dev/ocp-release:4.13.1-x86_64",
  "digest": "sha256:fakehash1234",
  "contentDigest": "sha256:fakehash1234",
  "listDigest": "",
  "config": {
    "id": "",
    "created": "2023-05-25T08:25:20Z",
    "container_config": {},
    "config": {
      "Env": [
        "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
        "container=oci",
        "GODEBUG=x509ignoreCN=0,madvdontneed=1",
        "__doozer=merge",
        "BUILD_RELEASE=202304211155.p0.ge08a279.assembly.stream",
        "BUILD_VERSION=v4.13.0",
        "OS_GIT_MAJOR=4",
        "OS_GIT_MINOR=13",
        "OS_GIT_PATCH=0",
        "OS_GIT_TREE_STATE=clean",
        "OS_GIT_VERSION=4.13.0-202304211155.p0.ge08a279.assembly.stream-e08a279",
        "SOURCE_GIT_TREE_STATE=clean",
        "OS_GIT_COMMIT=e08a279",
        "SOURCE_DATE_EPOCH=1681839693",
        "SOURCE_GIT_COMMIT=e08a27913bc6c6d0ca0663dc131d1183773cca10",
        "SOURCE_GIT_TAG=v1.0.0-1018-ge08a2791",
        "SOURCE_GIT_URL=https://github.com/openshift/cluster-version-operator"
      ],
      "Entrypoint": [
        "/usr/bin/cluster-version-operator"
      ],
      "Labels": {
        "io.openshift.release": "4.13.1",
        "io.openshift.release.base-image-digest": "sha256:fakehash1234"
      }
    },
    "architecture": "amd64",
    "size": 151043186,
    "rootfs": {
      "type": "layers",
      "diff_ids": [
        "sha256:fakehash1234",
        "sha256:fakehash1234",
        "sha256:fakehash1234",
        "sha256:fakehash1234",
        "sha256:fakehash1234",
        "sha256:fakehash1234"
      ]
    },
    "history": [
      {
        "created": "2023-05-25T08:25:20Z",
        "comment": "Release image for OpenShift"
      },
      {
        "created": "2023-05-25T08:25:20Z"
      },
      {
        "created": "2023-05-25T08:25:20Z"
      },
      {
        "created": "2023-05-25T08:25:20Z"
      },
      {
        "created": "2023-05-25T08:25:20Z"
      },
      {
        "created": "2023-05-25T08:25:20Z"
      }
    ],
    "os": "linux"
  },
  "metadata": {
    "kind": "cincinnati-metadata-v0",
    "version": "4.13.1",
    "previous": [
      "4.12.16",
      "4.12.17",
      "4.12.18",
      "4.12.19",
      "4.13.0"
    ],
    "metadata": {
      "url": "https://access.redhat.com/errata/RHSA-2023:3304"
    }
  },
  "references": {
    "kind": "ImageStream",
    "apiVersion": "image.openshift.io/v1",
    "metadata": {
      "name": "4.13.1",
      "creationTimestamp": "2023-05-25T08:25:20Z",
      "annotations": {
        "release.openshift.io/from-image-stream": "ocp/4.13-art-latest-2023-05-24-134135",
        "release.openshift.io/from-release": "registry.ci.openshift.org/ocp/release:4.13.0-0.nightly-2023-05-24-134135"
      }
    },
    "spec": {
      "lookupPolicy": {
        "local": false
      },
      "tags": [
        {
          "name": "machine-os-images",
          "annotations": {
            "io.openshift.build.commit.id": "b14856ffbc8fbe1f986cf1b7b835bbeaa786ce5e",
            "io.openshift.build.commit.ref": "",
            "io.openshift.build.source-location": "https://github.com/openshift/machine-os-images"
          },
          "from": {
            "kind": "DockerImage",
            "name": "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:fakehash1234"
          },
          "generation": null,
          "importPolicy": {},
          "referencePolicy": {
            "type": ""
          }
        }
      ]
    },
    "status": {
      "dockerImageRepository": ""
    }
  },
  "versions": null,
  "displayVersions": {
    "kubernetes": {
      "Version": "1.26.3",
      "DisplayName": ""
    },
    "machine-os": {
      "Version": "413.92.202305231734-0",
      "DisplayName": "Red Hat Enterprise Linux CoreOS"
    }
  },
  "images": null,
  "warnings": null
}
`
