#!/bin/bash -ex

# Create required directories
mkdir -p /home/var-lib-containers /var/lib/containers /home/var-tmp /var/tmp /home/fedora/.ssh

# Setup bind mounts
mount --bind /home/var-lib-containers /var/lib/containers
mount --bind /home/var-tmp /var/tmp
chmod 1777 /home/var-tmp /var/tmp
chown root:root /home/var-tmp /var/tmp
restorecon -r /var/lib/containers /var/tmp

# Configure fedora SSH access
chown -R fedora /home/fedora
sed -n 's,.*\(ssh-.*\s\),\1,p' /root/.ssh/authorized_keys > /home/fedora/.ssh/authorized_keys
chown fedora /home/fedora/.ssh/authorized_keys
chmod 600 /home/fedora/.ssh/authorized_keys
chmod 700 /home/fedora/.ssh
restorecon -r /home/fedora
