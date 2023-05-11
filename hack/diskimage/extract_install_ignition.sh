#!/usr/bin/env bash

# Extracts ignition config from boot partition

appliance=${1:-assets/appliance.qcow2}
target=${2:-assets/ignition/base}

mkdir -p $target
guestfish add $appliance : \
          run : \
          mount /dev/sda3 / : \
          copy-out /ignition/config.ign $target : \
          unmount-all : \
          exit
