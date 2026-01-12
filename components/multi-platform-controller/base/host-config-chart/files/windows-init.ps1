<powershell>
# ------------------
# Helper Functions
# ------------------
function Wait-Folder {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FolderPath,
        [Parameter(Mandatory=$false)]
        [int]$TimeoutSeconds = 30
    )
    Write-Host "Waiting for folder '${FolderPath}' to be created"
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    while (-not (Test-Path -Path ${FolderPath})) {
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

# ---------------------
# Docker Installation
# ---------------------
Invoke-WebRequest -UseBasicParsing "https://raw.githubusercontent.com/microsoft/Windows-Containers/Main/helpful_tools/Install-DockerCE/install-docker-ce.ps1" -o install-docker-ce.ps1
.\install-docker-ce.ps1

# ---------------------------------------------------
# OpenSSH Installation & Administrator Configuration
# ---------------------------------------------------
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Start-Service sshd
Set-Service -Name sshd -StartupType 'Automatic'

Remove-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue
New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' `
    -DisplayName 'OpenSSH Server (sshd)' `
    -Enabled True `
    -Direction Inbound `
    -Protocol TCP `
    -Action Allow `
    -LocalPort 22 `
    -RemoteAddress any | Out-Null

# Get public key from AWS Instance Metadata Service (IMDSv2)
$MAGIC_IP = "169.254.169.254"
$IMDS_TOKEN = Invoke-RestMethod -Uri "http://${MAGIC_IP}/latest/api/token" `
    -Method 'PUT' `
    -Headers @{'X-aws-ec2-metadata-token-ttl-seconds' = '21600'}
$PUBKEY = Invoke-RestMethod -Uri "http://${MAGIC_IP}/latest/meta-data/public-keys/0/openssh-key" `
    -Headers @{'X-aws-ec2-metadata-token' = $IMDS_TOKEN}

Start-Sleep 5

# Configure SSH authorized_keys for Administrator
$SSH_PATH = "C:\ProgramData\ssh"
Write-Host "Waiting for SSH Folder '${SSH_PATH}'"
if (-not (Wait-Folder -FolderPath ${SSH_PATH})) {
    Write-Warning "SSH folder not found, creating it manually"
    New-Item -ItemType Directory -Path $SSH_PATH | Out-Null
}
Write-Host "SSH Folder '${SSH_PATH}' found"

$PUBKEY | Out-File -FilePath "$SSH_PATH\administrators_authorized_keys" -Encoding ascii

$ACL = Get-Acl "$SSH_PATH\administrators_authorized_keys"
$Ar = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "NT AUTHORITY\SYSTEM",
    "FullControl",
    "Allow"
)
$ACL.SetAccessRule($Ar)
$Ar = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "BUILTIN\Administrators",
    "FullControl",
    "Allow"
)
$ACL.SetAccessRule($Ar)
Set-Acl "$SSH_PATH\administrators_authorized_keys" $ACL

Restart-Service sshd

# -------------------------------
# User Creation & Profile Setup
# -------------------------------
$user = "konflux-builder"
$password = $null
$userWasCreated = $false

# Create user if it doesn't exist
if ((Get-LocalUser -Name "${user}" -ErrorAction SilentlyContinue) -eq $null) {
    Write-Host "Creating user '${user}'"

    $passwordLength = 16
    $charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*-_=+'
    $rng = New-Object System.Security.Cryptography.RNGCryptoServiceProvider
    $bytes = New-Object byte[]($passwordLength)
    $rng.GetBytes($bytes)
    $password = -join ($bytes | ForEach-Object { $charset[$_ % $charset.Length] })
    $rng.Dispose()
    $securePassword = (ConvertTo-SecureString $password -AsPlainText -Force)

    New-LocalUser -Name $user -Password $securePassword -Description "Konflux Builder" | Out-Null
    Add-LocalGroupMember -Group 'Users' -Member $user
    Add-LocalGroupMember -Group 'OpenSSH Users' -Member $user

    $userWasCreated = $true
} else {
    Write-Host "User '${user}' already exists, skipping user creation"
}

# Create user profile
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
using System.Text;
public class UserEnv
{
    [DllImport("userenv.dll", SetLastError = true, CharSet = CharSet.Unicode)]
    public static extern int CreateProfile(
        [MarshalAs(UnmanagedType.LPWStr)] string pszUserSid,
        [MarshalAs(UnmanagedType.LPWStr)] string pszUserName,
        [MarshalAs(UnmanagedType.LPWStr)] StringBuilder pszProfilePath,
        uint cchProfilePath);
}
"@

$userSID = (New-Object System.Security.Principal.NTAccount($user)).Translate([System.Security.Principal.SecurityIdentifier]).Value
$profilePath = New-Object System.Text.StringBuilder(260)
$result = [UserEnv]::CreateProfile($userSID, $user, $profilePath, $profilePath.Capacity)

# Create user home directory
$userHome = "C:\Users\${user}"
Write-Host "Waiting for User Home '${userHome}' to be created"
if (-not (Wait-Folder -FolderPath ${userHome})) {
    Write-Error "Folder '${userHome}' not found! This may cause issues with SSH setup"
}

$hivePath = "C:\Users\${user}\NTUSER.DAT"
$loadedHive = "TempHive_${user}"

# Configure user registry permissions
try {
    if (Test-Path $hivePath) {
        $hiveLoaded = Test-Path "Registry::HKEY_USERS\${userSID}"

        if (-not $hiveLoaded) {
            & reg load "HKU\${loadedHive}" $hivePath 2>&1 | Out-Null
            Start-Sleep -Seconds 2
        } else {
            $loadedHive = $userSID
        }

        $registryPath = "Registry::HKEY_USERS\${loadedHive}"
        if (Test-Path $registryPath) {
            $acl = Get-Acl $registryPath

            $systemRule = New-Object System.Security.AccessControl.RegistryAccessRule(
                "NT AUTHORITY\SYSTEM",
                "FullControl",
                "ContainerInherit,ObjectInherit",
                "None",
                "Allow"
            )
            $acl.SetAccessRule($systemRule)

            $userRule = New-Object System.Security.AccessControl.RegistryAccessRule(
                $user,
                "FullControl",
                "ContainerInherit,ObjectInherit",
                "None",
                "Allow"
            )
            $acl.SetAccessRule($userRule)

            Set-Acl $registryPath $acl
        }

        if (-not $hiveLoaded) {
            [gc]::Collect()
            Start-Sleep -Seconds 2
            & reg unload "HKU\${loadedHive}" 2>&1 | Out-Null
        }
    }
} catch {
    Write-Warning "Failed to configure user registry permissions: $($_.Exception.Message)"
    Write-Warning "This may affect user profile settings, but SSH access should still work"
}

# ----------------------------------
# Docker User Access Configuration
# ----------------------------------
$dockerGroup = "docker-users"

if (-not (Get-LocalGroup -Name $dockerGroup -ErrorAction SilentlyContinue)) {
    New-LocalGroup -Name $dockerGroup -Description 'Docker Users' -ErrorAction SilentlyContinue
}

Add-LocalGroupMember -Group $dockerGroup -Member $user -ErrorAction SilentlyContinue

$dockerConfigPath = "C:\ProgramData\docker\config\daemon.json"
$dockerConfigDir = Split-Path $dockerConfigPath -Parent

if (-not (Test-Path $dockerConfigDir)) {
    New-Item -ItemType Directory -Path $dockerConfigDir -Force | Out-Null
}

$daemonConfig = @{ "group" = $dockerGroup }

# Configure Docker daemon to use the docker-users group
if (Test-Path $dockerConfigPath) {
    $existingConfig = Get-Content $dockerConfigPath -Raw | ConvertFrom-Json
    $existingConfig | Add-Member -NotePropertyName "group" -NotePropertyValue $dockerGroup -Force
    $existingConfig | ConvertTo-Json -Depth 10 | Set-Content $dockerConfigPath -Force
} else {
    $daemonConfig | ConvertTo-Json | Set-Content $dockerConfigPath -Force
}

if (Get-Service docker -ErrorAction SilentlyContinue) {
    Restart-Service docker -Force
}

# Add exclusion processes to Windows Defender
Add-MpPreference -ExclusionProcess "dockerd.exe" -ErrorAction SilentlyContinue
Add-MpPreference -ExclusionProcess "docker.exe" -ErrorAction SilentlyContinue
Add-MpPreference -ExclusionProcess "containerd.exe" -ErrorAction SilentlyContinue
Add-MpPreference -ExclusionProcess "vmcompute.exe" -ErrorAction SilentlyContinue

# -----------------------------
# Scoop Support Configuration
# -----------------------------
# Enable development mode - allows symbolic links to be created
Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock" `
    -Name "AllowDevelopmentWithoutDevLicense" `
    -Value 1 `
    -Type DWord `
    -Force

# Start the MSI server service - required for scoop installation
Set-Service -Name msiserver -StartupType Automatic -ErrorAction SilentlyContinue
Start-Service -Name msiserver -ErrorAction SilentlyContinue

# --------------------------------
# User SSH Key & Directory Setup
# --------------------------------
$tempKey = "${env:TEMP}\${user}"

# Remove existing SSH keys if they exist
if (Test-Path "${tempKey}") { Remove-Item -Force "${tempKey}" }
if (Test-Path "${tempKey}.pub") { Remove-Item -Force "${tempKey}.pub" }

ssh-keygen -t rsa -f "${tempKey}" -N `"`" | Out-Null

$privateKeyPath = "C:\Users\Administrator\${user}"
mv "${env:TEMP}\${user}" "${privateKeyPath}"

# Configure ACL for the private key
$ACL = Get-Acl "${privateKeyPath}"
$Ar = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "NT AUTHORITY\SYSTEM",
    "FullControl",
    "Allow"
)
$ACL.SetAccessRule($Ar)
$Ar = New-Object System.Security.AccessControl.FileSystemAccessRule(
    "BUILTIN\Administrators",
    "FullControl",
    "Allow"
)
$ACL.SetAccessRule($Ar)
Set-Acl "${privateKeyPath}" ${ACL}

$publicKeyTempPath = "C:\Users\Public\${user}.pub"
Copy-Item "${tempKey}.pub" $publicKeyTempPath -Force
Start-Sleep 10

# Create  additional user directories
$sshPath = Join-Path $userHome ".ssh"
$buildPath = Join-Path $userHome "build"
$scoopPath = Join-Path $userHome "scoop"
$authorizedKeysPath = Join-Path $sshPath "authorized_keys"

New-Item -ItemType Directory -Force -Path $sshPath | Out-Null
New-Item -ItemType Directory -Force -Path $buildPath | Out-Null
New-Item -ItemType Directory -Force -Path $scoopPath | Out-Null

if (Test-Path $publicKeyTempPath) {
    Copy-Item $publicKeyTempPath $authorizedKeysPath -Force
}

# Set owner and permissions for additional user directories
$userSidObj = New-Object System.Security.Principal.SecurityIdentifier($userSID)
foreach ($path in @($sshPath, $buildPath, $scoopPath, $authorizedKeysPath)) {
    if (Test-Path $path) {
        $acl = Get-Acl $path
        $acl.SetOwner($userSidObj)
        Set-Acl $path $acl
    }
}

# Configure ACL for the SSH directory
if (Test-Path $sshPath) {
    $sshAcl = Get-Acl $sshPath
    $sshAcl.SetAccessRuleProtection($true, $false)
    $sshAcl.Access | ForEach-Object { $sshAcl.RemoveAccessRule($_) } | Out-Null

    $userRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        $user,
        "FullControl",
        "ContainerInherit,ObjectInherit",
        "None",
        "Allow"
    )
    $sshAcl.SetAccessRule($userRule)

    $systemRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        "NT AUTHORITY\SYSTEM",
        "FullControl",
        "ContainerInherit,ObjectInherit",
        "None",
        "Allow"
    )
    $sshAcl.SetAccessRule($systemRule)

    Set-Acl $sshPath $sshAcl
}

if (Test-Path $authorizedKeysPath) {
    $keyAcl = Get-Acl $authorizedKeysPath
    $keyAcl.SetAccessRuleProtection($true, $false)
    $keyAcl.Access | ForEach-Object { $keyAcl.RemoveAccessRule($_) } | Out-Null

    $userKeyRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        $user,
        "Read",
        "Allow"
    )
    $keyAcl.SetAccessRule($userKeyRule)

    $systemKeyRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        "NT AUTHORITY\SYSTEM",
        "FullControl",
        "Allow"
    )
    $keyAcl.SetAccessRule($systemKeyRule)

    Set-Acl $authorizedKeysPath $keyAcl
}

if (Test-Path $publicKeyTempPath) {
    Remove-Item -Path $publicKeyTempPath -Force -ErrorAction SilentlyContinue
}
if (Test-Path "${tempKey}.pub") {
    Remove-Item -Force "${tempKey}.pub"
}

# --------------------------
# Scoop Installation Script
# --------------------------
if ($userWasCreated) {
    $scoopScript = @"
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
Write-Host "Installing Scoop..."
Invoke-RestMethod -Uri 'https://get.scoop.sh' | Invoke-Expression
scoop config use_lessmsi true
Write-Host "Scoop installed successfully!"
"@
    $scoopScriptPath = "C:\Users\${user}\install_scoop.ps1"
    $scoopScript | Out-File -FilePath $scoopScriptPath -Encoding UTF8 -Force

    $scriptAcl = Get-Acl $scoopScriptPath
    $scriptAcl.SetOwner($userSidObj)
    Set-Acl $scoopScriptPath $scriptAcl
}
</powershell>
<persist>true</persist>
