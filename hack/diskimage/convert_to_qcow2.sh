#!/usr/bin/env bash

# Converts appliance diskimage to qcow2 for allowing snapshot creation

source=${1:-assets/appliance.raw}
target=${2:-assets/appliance.qcow2}

qemu-img convert -f raw -O qcow2 "$source" "$target"
