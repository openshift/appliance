#!/bin/bash
# Update rpm-prefetching/rpms.lock.yaml using rpm-lockfile-prototype.
#
# This script generates the RPM lockfile from rpms.in.yaml using
# rpm-lockfile-prototype inside a UBI9 minimal container.

set -e

# Get script directory and repo root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
RPM_PREFETCH_DIR="${REPO_ROOT}/rpm-prefetching"
INPUT_FILE="${RPM_PREFETCH_DIR}/rpms.in.yaml"
OUTPUT_FILE="${RPM_PREFETCH_DIR}/rpms.lock.yaml"
DOCKERFILE="${REPO_ROOT}/Dockerfile.openshift-appliance.ds"

# Check if input file exists
if [ ! -f "${INPUT_FILE}" ]; then
    echo "Error: Input file not found: ${INPUT_FILE}" >&2
    exit 1
fi

# Check if Dockerfile exists
if [ ! -f "${DOCKERFILE}" ]; then
    echo "Error: Dockerfile not found: ${DOCKERFILE}" >&2
    exit 1
fi

# Prompt for RH_USER if empty
if [ -z "${RH_USER}" ]; then
    read -p "Enter Red Hat username: " RH_USER
    if [ -z "${RH_USER}" ]; then
        echo "Error: Red Hat username cannot be empty" >&2
        exit 1
    fi
fi

# Prompt for password
read -s -p "Enter password for ${RH_USER}: " PASSWORD
echo ""
if [ -z "${PASSWORD}" ]; then
    echo "Error: Password cannot be empty" >&2
    exit 1
fi

# Require docker
if command -v docker >/dev/null 2>&1; then
    CONTAINER_CMD="docker"
else
    echo "Error: docker not found in PATH" >&2
    exit 1
fi

# Build subscription-manager register command
SUB_MGR_CMD="subscription-manager register --username=\${RH_USER} --password=\${PASSWORD}"

echo "Using ${CONTAINER_CMD} to update RPM lockfile..."
echo "Input:  ${INPUT_FILE}"
echo "Output: ${OUTPUT_FILE}"
echo ""

# Run the container from the repo root so paths work correctly
cd "${REPO_ROOT}"

# Build container command arguments
# Force x86_64 platform to ensure consistent repo configuration across host architectures
CONTAINER_ARGS=(
    "run" "--rm" "-it"
    "--platform" "linux/amd64"
    "-v" "$(pwd):/source:Z"
)

# Add environment variables
CONTAINER_ARGS+=("-e" "RH_USER=${RH_USER}")
CONTAINER_ARGS+=("-e" "PASSWORD=${PASSWORD}")

# Add image
CONTAINER_ARGS+=("registry.access.redhat.com/ubi9")

# Build the bash command to run inside the container
BASH_CMD=$(cat <<EOF
${SUB_MGR_CMD}
subscription-manager refresh
dnf install -y pip skopeo git
pip install --user git+https://github.com/konflux-ci/rpm-lockfile-prototype.git
skopeo login registry.redhat.io -u \$RH_USER -p \$PASSWORD
subscription-manager repos --enable codeready-builder-for-rhel-9-x86_64-rpms
/usr/bin/cp -f /etc/yum.repos.d/redhat.repo /source/rpm-prefetching/redhat.repo
cd /source
~/.local/bin/rpm-lockfile-prototype rpm-prefetching/rpms.in.yaml --outfile rpm-prefetching/rpms.lock.yaml
rm -rf /source/rpm-prefetching/redhat.repo
EOF
)

# Run the container with all setup and execution commands
"${CONTAINER_CMD}" "${CONTAINER_ARGS[@]}" "bash" "-c" "${BASH_CMD}"

# Verify the output file was created and is not empty
if [ ! -s "${OUTPUT_FILE}" ]; then
    echo "Error: Output file is empty or was not created" >&2
    exit 1
fi

echo "Successfully updated ${OUTPUT_FILE}"

