#!/usr/bin/env bash

# Embeds install ingition config in boot partition of the diskimage

appliance=${1:-appliance.qcow2}
snapshot=${2:-assets/snapshot.qcow2}
ignition=${3:-assets/ignition/install/config.ign}

qemu-img create -f qcow2 -b $appliance -F qcow2 $snapshot
guestfish add $snapshot : \
          run : \
          mount /dev/sda3 / : \
          copy-in "$ignition" /ignition/ : \
          unmount-all : \
          exit
