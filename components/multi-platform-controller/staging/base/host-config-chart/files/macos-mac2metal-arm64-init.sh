#!/bin/bash
set -eu
set -x

user="konflux-builder"

# Check if user already exists
if ! id "$user" &>/dev/null; then
    # Generate random password
    random_password=$(openssl rand -base64 32)

    # Create user
    sudo sysadminctl -addUser "$user" -fullName "Konflux Builder" -password "$random_password" -home /Users/$user

    # Clear password from variable
    unset random_password
else
    echo "User $user already exists, skipping user creation"
fi

# Create home directory if it doesn't exist
sudo mkdir -p /Users/$user

# Create SSH directory
sudo mkdir -p /Users/$user/.ssh

# Remove existing SSH keys if they exist
sudo rm -f /Users/$user/.ssh/id_rsa /Users/$user/.ssh/id_rsa.pub

# Generate new SSH keys
sudo ssh-keygen -t rsa -b 4096 -f /Users/$user/.ssh/id_rsa -N "" -C ""

# Set proper permissions on .ssh directory
sudo chmod 700 /Users/$user/.ssh

# Create/overwrite authorized_keys
sudo chmod 600 /Users/$user/.ssh/authorized_keys 2>/dev/null || true
sudo cat /Users/$user/.ssh/id_rsa.pub | sudo tee /Users/$user/.ssh/authorized_keys > /dev/null
sudo cat /Users/$user/.ssh/id_rsa | sudo tee /Users/ec2-user/$user > /dev/null

# Set ownership of entire home directory to ensure user has full control
sudo chown -R $user:staff /Users/$user

# Set ownership of the copied private key to ec2-user
sudo chown ec2-user:staff /Users/ec2-user/$user
sudo chmod 600 /Users/ec2-user/$user
