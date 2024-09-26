#!/usr/bin/env bash

# Enable debugging
set -eux

DEVICE="$( ls /dev/disk/by-id/nvme-Amazon_EC2_NVMe_Instance_Storage_* | sort | head -n1 )"
echo "Going to use device $DEVICE"

export MOUNT_POINT="/var/home/nvme"
export TARGET_DIR="/var/lib"
export FSTYPE="xfs"

mkdir -p ${MOUNT_POINT}

if lsblk -no FSTYPE "$DEVICE" | grep -qE '\S'; then
    echo "File system already exists on $DEVICE."
else
    echo "No file system found on $DEVICE. Creating XFS filesystem..."
    mkfs -t "$FSTYPE" "$DEVICE"
fi

if ! grep -q "$DEVICE $MOUNT_POINT" /etc/fstab; then
    echo "$DEVICE $MOUNT_POINT $FSTYPE defaults 0 0" >> /etc/fstab
fi

mount ${MOUNT_POINT}

mkdir -p ${MOUNT_POINT}/var-lib-kubelet-pods
mount --bind ${MOUNT_POINT}/var-lib-kubelet-pods ${TARGET_DIR}/kubelet/pods

mkdir -p ${MOUNT_POINT}/var-lib-containers
mount --bind ${MOUNT_POINT}/var-lib-containers ${TARGET_DIR}/containers

restorecon -R -v -F /var/lib/kubelet/pods /var/lib/containers

echo "Filesystem setup and mounting complete."
