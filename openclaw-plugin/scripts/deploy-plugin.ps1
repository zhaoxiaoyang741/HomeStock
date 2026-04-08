[CmdletBinding()]
param(
  [string]$Host = '192.168.174.131',
  [string]$User = 'claw',
  [string]$RemoteDir = '/home/claw/tmp',
  [string]$PackageName = 'home-inventory-plugin-1.0.0.tgz'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Require-Command {
  param([string]$Name)

  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "Missing required command: $Name"
  }
}

function ConvertTo-BashSingleQuoted {
  param([string]$Value)

  return "'" + ($Value -replace "'", "'""'""'") + "'"
}

function Invoke-External {
  param(
    [string]$Description,
    [string]$Command,
    [string[]]$Arguments
  )

  Write-Host "==> $Description"
  & $Command @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "$Description failed with exit code $LASTEXITCODE"
  }
}

function Ensure-SshKey {
  $sshDir = Join-Path $HOME '.ssh'
  $privateKeyPath = Join-Path $sshDir 'id_ed25519'
  $publicKeyPath = "${privateKeyPath}.pub"

  if (-not (Test-Path $privateKeyPath) -or -not (Test-Path $publicKeyPath)) {
    if (-not (Test-Path $sshDir)) {
      New-Item -ItemType Directory -Path $sshDir | Out-Null
    }

    Invoke-External `
      -Description 'Generating SSH key pair' `
      -Command 'ssh-keygen' `
      -Arguments @('-t', 'ed25519', '-N', '', '-f', $privateKeyPath)
  }

  return @{
    PrivateKey = $privateKeyPath
    PublicKey = $publicKeyPath
  }
}

function Test-PasswordlessLogin {
  param(
    [string]$Target,
    [string]$PrivateKeyPath
  )

  & ssh -o BatchMode=yes -o ConnectTimeout=5 -i $PrivateKeyPath $Target 'true' | Out-Null
  return $LASTEXITCODE -eq 0
}

function Install-PublicKey {
  param(
    [string]$Target,
    [string]$PrivateKeyPath,
    [string]$PublicKeyPath
  )

  $publicKey = (Get-Content -Raw -Path $PublicKeyPath).Trim()
  $escapedKey = ConvertTo-BashSingleQuoted -Value $publicKey
  $remoteCommand = @(
    'set -e',
    'mkdir -p ~/.ssh',
    'chmod 700 ~/.ssh',
    'touch ~/.ssh/authorized_keys',
    'chmod 600 ~/.ssh/authorized_keys',
    "grep -qxF $escapedKey ~/.ssh/authorized_keys || echo $escapedKey >> ~/.ssh/authorized_keys"
  ) -join '; '

  Write-Host '==> Installing local public key on remote host'
  Write-Host '    The first run may prompt once for the remote account password.'
  & ssh -i $PrivateKeyPath $Target "bash -lc $(ConvertTo-BashSingleQuoted -Value $remoteCommand)"
  if ($LASTEXITCODE -ne 0) {
    throw "Installing SSH public key failed with exit code $LASTEXITCODE"
  }
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$pluginDir = Split-Path -Parent $scriptDir
$packagePath = Join-Path $pluginDir $PackageName
$target = "$User@$Host"
$remotePackagePath = "$RemoteDir/$PackageName" -replace '//+', '/'

if (-not (Test-Path $packagePath)) {
  throw "Package not found: $packagePath"
}

Require-Command -Name 'ssh'
Require-Command -Name 'scp'
Require-Command -Name 'ssh-keygen'

$sshKey = Ensure-SshKey

if (-not (Test-PasswordlessLogin -Target $target -PrivateKeyPath $sshKey.PrivateKey)) {
  Install-PublicKey -Target $target -PrivateKeyPath $sshKey.PrivateKey -PublicKeyPath $sshKey.PublicKey

  if (-not (Test-PasswordlessLogin -Target $target -PrivateKeyPath $sshKey.PrivateKey)) {
    throw 'Passwordless SSH login is still unavailable after installing the public key.'
  }
}

Invoke-External `
  -Description "Ensuring remote directory $RemoteDir exists" `
  -Command 'ssh' `
  -Arguments @('-i', $sshKey.PrivateKey, $target, "mkdir -p $RemoteDir")

Invoke-External `
  -Description "Copying $PackageName to ${target}:$RemoteDir" `
  -Command 'scp' `
  -Arguments @('-i', $sshKey.PrivateKey, $packagePath, "${target}:${RemoteDir}/")

$deployCommand = @(
  'set -e',
  "printf 'Y\n' | openclaw plugins uninstall home-inventory",
  "openclaw plugins install '$remotePackagePath'",
  'openclaw gateway restart'
) -join '; '

Invoke-External `
  -Description 'Deploying plugin and restarting OpenClaw gateway' `
  -Command 'ssh' `
  -Arguments @('-i', $sshKey.PrivateKey, $target, "bash -lc $(ConvertTo-BashSingleQuoted -Value $deployCommand)")
