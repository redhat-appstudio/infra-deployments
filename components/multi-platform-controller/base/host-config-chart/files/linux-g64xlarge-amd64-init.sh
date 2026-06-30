#!/bin/bash

set -xeuo pipefail

configure_nvidia_cdi() {
  # generate Nvdia CDI with retry
  for i in {1..10}; do
    su - ec2-user -c 'nvidia-ctk cdi generate --output=/etc/cdi/nvidia.yaml' 
    su - ec2-user -c 'nvidia-ctk cdi generate --output=/var/run/cdi/nvidia.yaml'

    # expect nvidia.com/gpu=all device to be in the generated list
    if nvidia-ctk cdi list 2>/dev/null | grep -q 'nvidia.com/gpu=all'; then
      echo "Nvidia CDI Ready"
      set -e
      return 0
    fi

    # sleep only if we'll be retrying again
    [ "${i}" -lt "10" ] && sleep 1
  done

  echo "Nvidia CDI Failed"
  set -e
  return 1
}

# Format and mount NVMe disk
mkfs -t xfs /dev/nvme1n1
mount /dev/nvme1n1 /home

# Create required directories
mkdir -p /home/var-lib-containers /var/lib/containers /home/var-tmp /var/tmp /home/ec2-user/.ssh

# Setup bind mounts
mount --bind /home/var-lib-containers /var/lib/containers
mount --bind /home/var-tmp /var/tmp
chmod 1777 /home/var-tmp /var/tmp
chown root:root /home/var-tmp /var/tmp
restorecon -r /var/lib/containers /var/tmp

# GPU setup
mkdir -p /etc/cdi /var/run/cdi
chmod a+rwx /etc/cdi /var/run/cdi
setsebool container_use_devices 1 2>/dev/null || true
configure_nvidia_cdi
chmod a+rw /etc/cdi/nvidia.yaml
chmod a+rw /var/run/cdi/nvidia.yaml

# Configure ec2-user SSH access
chown -R ec2-user /home/ec2-user
sed -n 's,.*\(ssh-.*\s\),\1,p' /root/.ssh/authorized_keys > /home/ec2-user/.ssh/authorized_keys
chown ec2-user /home/ec2-user/.ssh/authorized_keys
chmod 600 /home/ec2-user/.ssh/authorized_keys
chmod 700 /home/ec2-user/.ssh
restorecon -r /home/ec2-user
