# ac-no-external-dependencies acceptance criteria

The iso-builder is distributed as a single executable file. It must not depend on
external CLI tools being present on the host system at runtime. Each scenario below
addresses the removal of a specific external dependency without introducing any new
ones.

These acceptance criteria apply exclusively to the iso-builder code path. The
existing live ISO code flows may continue to use their current external dependencies 
(e.g. podman, skopeo, coreos-installer) unchanged.

## oc client

# scenario 1: Acquire oc client autonomously
Given an iso-builder instance with an embedded configuration referencing an OCP release
When the build command is executed on a host that does not have oc installed
Then the iso-builder downloads the appropriate oc binary for the host platform and architecture from mirror.openshift.com
And uses the downloaded oc binary for all subsequent operations
And does not require oc to be pre-installed on the host

## genisoimage

# scenario 2: Build ISO without genisoimage
Given an iso-builder instance with an embedded configuration
When the build command is executed on a host that does not have genisoimage installed
Then the iso-builder creates the ISO image using the go-diskfs Go library
And does not shell out to genisoimage or any other external ISO authoring tool
And the resulting ISO is a valid El Torito bootable ISO image

## distribution registry

# scenario 3: Run distribution registry without podman
Given an iso-builder instance that needs a local distribution registry during the build
When the build command is executed on a host that does not have podman installed
Then the iso-builder extracts the distribution registry binary from the registry container image
And runs the registry binary directly as a host process
And does not require podman, docker, or any other container runtime on the host

## isohybrid

# scenario 4: Build ISO without isohybrid on x86-64
Given an iso-builder instance with an embedded configuration targeting x86-64
When the build command is executed on a host that does not have isohybrid installed
Then the iso-builder produces a valid El Torito bootable ISO image without running isohybrid
And prints a message informing the user that if USB booting is required they should run isohybrid on the ISO before writing it to a USB disk

# scenario 5: Preserve hybrid ISO for existing appliance builds
Given the existing appliance builder (not the standalone iso-builder)
When it produces an OVE ISO image on x86-64
Then the resulting ISO remains hybrid (El Torito + MBR) as it is today
And isohybrid continues to be applied in the appliance builder code path

## split

# scenario 6: Split registry data image without coreutils split
Given a registry data image that exceeds 4 GiB
When the iso-builder packages the registry data into the outer ISO
Then it splits the data into chunks smaller than 4 GiB using native Go I/O
And does not shell out to the GNU coreutils split command or any other external tool

## openshift-installer

# scenario 7: Extract openshift-install from the release image
Given an iso-builder instance with an embedded configuration referencing an OCP release
When the build command needs the unconfigured ignition for the ISO
Then the iso-builder extracts the openshift-install binary from the release image in the local registry using oc adm release extract
And uses the extracted openshift-install to generate the unconfigured ignition
And does not require openshift-install to be pre-installed on the host

## oc-mirror

# scenario 8: Download oc-mirror autonomously
Given an iso-builder instance that needs to mirror the OCP release to local registry storage
When the build command is executed on a host that does not have oc-mirror installed
Then the iso-builder downloads the correct oc-mirror binary for the host platform and architecture
And uses it to mirror the release image to the local registry storage
And does not require oc-mirror to be pre-installed on the host

## skopeo

# scenario 9: Eliminate skopeo without adding new external dependencies
Given an iso-builder instance that needs to save a container image in containers/storage format
When the build command is executed on a host that does not have skopeo installed
Then the iso-builder performs the image save operation using vendored Go libraries
And does not shell out to skopeo or any other external tool
And does not introduce a new runtime dependency on podman or any other external binary

## coreos-installer

# scenario 10: Embed ignition without coreos-installer
Given an iso-builder instance that needs to embed ignition data into a CoreOS ISO
When the build command is executed on a host that does not have coreos-installer installed
Then the iso-builder embeds the ignition using the assisted-image-service Go library
And does not shell out to coreos-installer or any other external tool

## general

# scenario 11: No new external runtime dependencies
Given any change made to eliminate an external dependency
When the change is implemented
Then it does not introduce any new external runtime binary dependency
And the iso-builder remains distributable as a single executable file

# scenario 12: Scope limited to iso-builder code path
Given a change made to eliminate an external dependency for the iso-builder
When the change is implemented
Then it does not modify the existing live ISO code flows
And those code flows continue to work with their current external dependencies

# scenario 13: Use current upstream Go module paths
Given any Go library dependency added or updated in the iso-builder
When the dependency is a container ecosystem library (e.g. image, storage)
Then it uses the current canonical module path (e.g. go.podman.io/image/v5, go.podman.io/storage)
And does not use deprecated module paths (e.g. github.com/containers/image/v5, github.com/containers/storage)
