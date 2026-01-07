<powershell>
function Wait-Folder {
  param(
      [Parameter(Mandatory=$true)]
      [string]$FolderPath,

      [Parameter(Mandatory=$false)]
      [int]$TimeoutSeconds = 30
  )
  Write-Host "Waiting for folder '${FolderPath}' to be created"

  # Start a timer
  $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

  while (-not (Test-Path -Path ${FolderPath})) {
    # Check if we have exceeded the timeout
    if ($stopwatch.Elapsed.TotalSeconds -ge $TimeoutSeconds) {
      Write-Error "Timeout reached! Folder was not created within $TimeoutSeconds seconds."
     $stopwatch.Stop()
      return $false
    }

    Write-Host "Waiting for folder..." -NoNewline
    Start-Sleep -Seconds 1
  }
  $stopwatch.Stop()

  return $true
}

## -------------------------------------
## --------- Create Local User ---------
## -------------------------------------
$user = "konflux-builder"
if ((Get-LocalUser -Name "${user}" -ErrorAction SilentlyContinue) -eq $null) {
  $password = (-join([char[]](33..122) | Get-Random -Count 30))
  $securePassword = (ConvertTo-SecureString $password -AsPlainText -Force)

  # Create user
  New-LocalUser -Name $user -Password $securePassword -Description "Konflux Builder" | Out-Null
  Add-LocalGroupMember -Group 'Users' -Member "${user}"
  Add-LocalGroupMember -Group 'OpenSSH Users' -Member "${user}"

  # Create a Credential Object for the new user
  $userCred = New-Object System.Management.Automation.PSCredential($user, $securePassword)

  # Start a dummy Process as the new User
  # This is required to have the user home folder initialized.
  # TODO: can we do better?
  Start-Process -FilePath "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" `
                -Credential ${userCred} `
                -ArgumentList "-Command exit" `
                -LoadUserProfile `
                -WindowStyle Hidden `
                -WorkingDirectory "C:\Users\" `
                -Wait

  # Create a Key to login as user
  Write-Host "Creating SSH Key for user '${user}'"
  $tempKey = "${env:TEMP}\${user}"
  if (Test-Path "${tempKey}") { Remove-Item -Force "${tempKey}" }
  if (Test-Path "${tempKey}.pub") { Remove-Item -Force "${tempKey}.pub" }
  ssh-keygen -t rsa  -f "${tempKey}" -N `"`" | Out-Null

  # Move private key to a secure location and restrict access to it
  $privateKeyPath = "C:\Users\Administrator\${user}"
  mv "${env:TEMP}\${user}" "${privateKeyPath}"
  $ACL = Get-Acl "${privateKeyPath}"
  $Ar = New-Object System.Security.AccessControl.FileSystemAccessRule("NT AUTHORITY\SYSTEM", "FullControl", "Allow")
  $ACL.SetAccessRule($Ar)
  $Ar = New-Object System.Security.AccessControl.FileSystemAccessRule("BUILTIN\Administrators", "FullControl", "Allow")
  $ACL.SetAccessRule($Ar)
  Set-Acl "${privateKeyPath}" ${ACL}

  # Init home folder
  $userHome = "C:\Users\${user}"
  Write-Host "Waiting for User Home '${userHome}' to be created"

  # Ensure User's home folder is eventually created
  if (-not (Wait-Folder -FolderPath ${userHome})) {
    Write-Error "Folder '${userHome}' not found! Cleanup..." -ForegroundColor Red
    if (Test-Path "\${tempKey}") { Remove-Item -Force "${tempKey}" }
    if (Test-Path "\${tempKey}.pub") { Remove-Item -Force "${tempKey}.pub" }
    exit 1
  }

  # Set-up SSH Keys for User
  Write-Host "User Home found. Configuring SSH access" -ForegroundColor Green
  New-Item -ItemType Directory -Force -Path "${userHome}\.ssh"
  New-Item -ItemType Directory -Force -Path "${userHome}\build"

  # Copying and removing to preserve file permissions! Do not use `mv`! :)
  cp "${tempKey}.pub" "${userHome}\.ssh\authorized_keys"
  rm "${tempKey}.pub"
}

## ---------------------------------------------
## --------- Enable Windows Containers ---------
## ---------------------------------------------
Invoke-WebRequest -UseBasicParsing "https://raw.githubusercontent.com/microsoft/Windows-Containers/Main/helpful_tools/Install-DockerCE/install-docker-ce.ps1" -o install-docker-ce.ps1
.\install-docker-ce.ps1 -NoRestart
if (${global:RebootRequired}) {
  Restart-Computer
  exit
}

# Create docker-users group and add konflux-builder to it
if ((Get-LocalGroup -Name 'docker-users') -eq $null) {
  New-LocalGroup -Name 'docker-users' -Description 'Docker Users'
}
if ((Get-LocalGroupMember -Group 'docker-users' -Member 'konflux-builder') -eq $null) {
  Add-LocalGroupMember -Group 'docker-users' -Member "${user}"
}

# allow the docker-users group to use docker
$dockerConfigPath = "C:\ProgramData\docker\config\daemon.json"
$existingConfig = Get-Content $dockerConfigPath -Raw | ConvertFrom-Json
if ((${existingConfig}.group) -eq $null) {
  $existingConfig | Add-Member -NotePropertyName "group" -NotePropertyValue "docker-users" -Force
  $existingConfig | ConvertTo-Json -Depth 10 | Set-Content $dockerConfigPath
  Restart-Service docker
}

# Exclude docker in Windows Defender
Add-MpPreference -ExclusionProcess "dockerd.exe"
Add-MpPreference -ExclusionProcess "docker.exe"
Add-MpPreference -ExclusionProcess "containerd.exe"
Add-MpPreference -ExclusionProcess "vmcompute.exe"

## -------------------------------------
## --------- Configure OpenSSH ---------
## -------------------------------------

# Install OpenSSH Server
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0

# Start the sshd service and set it to start automatically
Start-Service sshd
Set-Service -Name sshd -StartupType 'Automatic'

# Grab the Public Key from AWS Metadata and configure authorized_keys
# This allows you to log in with your .pem/.ppk file instead of a password
$MAGIC_IP = "169.254.169.254"
$IMDS_TOKEN = Invoke-RestMethod -Uri "http://${MAGIC_IP}/latest/api/token" -Method 'PUT' -Headers @{'X-aws-ec2-metadata-token-ttl-seconds' = '21600'}
$PUBKEY = Invoke-RestMethod -Uri "http://${MAGIC_IP}/latest/meta-data/public-keys/0/openssh-key" -Headers @{'X-aws-ec2-metadata-token' = $IMDS_TOKEN}

# Ensure SSH_PATH folder was created
$SSH_PATH = "C:\ProgramData\ssh"
Write-Host "Waiting for SSH Folder"
if (-not (Wait-Folder -FolderPath ${SSH_PATH})) {
  Write-Error "Folder '${SSH_PATH}' not found! Exiting..." -ForegroundColor Red
  exit 1
}
Write-Host "Folder '${SSH_PATH}' found"

# Add key to administrators_authorized_keys
$PUBKEY | Out-File -FilePath "$SSH_PATH\administrators_authorized_keys" -Encoding ascii

# Fix permissions (ACLs) for the authorized_keys file
# OpenSSH is strict: only System and Administrators should have access
$ACL = Get-Acl "$SSH_PATH\administrators_authorized_keys"
$Ar = New-Object System.Security.AccessControl.FileSystemAccessRule("NT AUTHORITY\SYSTEM", "FullControl", "Allow")
$ACL.SetAccessRule($Ar)
$Ar = New-Object System.Security.AccessControl.FileSystemAccessRule("BUILTIN\Administrators", "FullControl", "Allow")
$ACL.SetAccessRule($Ar)
Set-Acl "$SSH_PATH\administrators_authorized_keys" $ACL

# Restart sshd to apply key changes
Restart-Service sshd

# Configure the Firewall to allow SSH (Port 22)
Remove-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' | Out-Null
New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22 -RemoteAddress any
Write-Output "Firewall Rule 'OpenSSH-Server-In-TCP' updated"
</powershell>
<persist>true</persist>

