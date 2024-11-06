#!/usr/bin/env bash

# Enable debugging
set -eux

if [ -f /host-etc-systemd-dir/nvme-init-done-2 ]; then
    echo 'NVMe init already done'
    oc adm uncordon $NODE_NAME
    exit 0
fi

cp /scripts/mount.sh /host-usr-local-bin/mount.sh
chmod 0755 /host-usr-local-bin/mount.sh

cp /scripts/nvme-storage.service /host-etc-systemd-dir/nvme-storage.service
chmod 0644 /host-etc-systemd-dir/nvme-storage.service

oc adm cordon $NODE_NAME
oc adm drain $NODE_NAME --delete-emptydir-data --ignore-daemonsets --grace-period=-1

nsenter -t 1 -m -u -i -n -p -- systemctl daemon-reload
nsenter -t 1 -m -u -i -n -p -- systemctl enable nvme-storage.service
touch /host-etc-systemd-dir/nvme-init-done
nsenter -t 1 -m -u -i -n -p -- systemctl reboot
