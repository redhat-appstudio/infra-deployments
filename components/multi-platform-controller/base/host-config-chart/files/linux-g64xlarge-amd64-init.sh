Content-Type: multipart/mixed; boundary="//"
MIME-Version: 1.0

--//
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config.txt"

#cloud-config
cloud_final_modules:
  - [scripts-user, always]

--//
Content-Type: text/x-shellscript; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="userdata.txt"

#!/bin/bash -ex

# Format and mount NVMe disk
mkfs -t xfs /dev/nvme1n1
mount /dev/nvme1n1 /home

# Create required directories
mkdir -p /home/var-lib-containers /var/lib/containers /home/var-tmp /var/tmp /home/ec2-user/.ssh

# Setup bind mounts
mount --bind /home/var-lib-containers /var/lib/containers
mount --bind /home/var-tmp /var/tmp
chmod a+rw /var/tmp
restorecon -r /var/lib/containers /var/tmp

# Configure ec2-user SSH access
chown -R ec2-user /home/ec2-user
sed -n 's,.*\(ssh-.*\s\),\1,p' /root/.ssh/authorized_keys > /home/ec2-user/.ssh/authorized_keys
chown ec2-user /home/ec2-user/.ssh/authorized_keys
chmod 600 /home/ec2-user/.ssh/authorized_keys
chmod 700 /home/ec2-user/.ssh
restorecon -r /home/ec2-user

# GPU setup
mkdir -p /etc/cdi /var/run/cdi
chmod a+rwx /etc/cdi /var/run/cdi
setsebool container_use_devices 1 2>/dev/null || true
su - ec2-user
nvidia-ctk cdi generate --output=/etc/cdi/nvidia.yaml
chmod a+rw /etc/cdi/nvidia.yaml
--//--
