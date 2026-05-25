[CmdletBinding()]
param(
  [string]$EnvFile = ".env.timeweb",
  [switch]$PackageOnly,
  [switch]$NoDockerSshpass
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

function Import-DotEnv {
  param([string]$Path)

  $values = @{}
  if (-not (Test-Path -LiteralPath $Path)) {
    return $values
  }

  Get-Content -LiteralPath $Path -Encoding UTF8 | ForEach-Object {
    $line = $_.Trim()
    if ($line.Length -eq 0 -or $line.StartsWith("#") -or -not $line.Contains("=")) {
      return
    }

    $key, $value = $line.Split("=", 2)
    $key = $key.Trim()
    $value = $value.Trim()

    if ($value.Length -ge 2 -and (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'")))) {
      $value = $value.Substring(1, $value.Length - 2)
    }

    if ($key -match "^[A-Za-z_][A-Za-z0-9_]*$") {
      $values[$key] = $value
      [Environment]::SetEnvironmentVariable($key, $value, "Process")
    }
  }

  return $values
}

function Merge-Hashtable {
  param(
    [hashtable]$Target,
    [hashtable]$Source
  )

  foreach ($key in $Source.Keys) {
    $Target[$key] = $Source[$key]
  }
}

function Remote-Quote {
  param([string]$Value)
  return "'" + $Value.Replace("'", "'\''") + "'"
}

$config = @{}
Merge-Hashtable $config (Import-DotEnv (Join-Path $repoRoot $EnvFile))
Merge-Hashtable $config (Import-DotEnv (Join-Path $repoRoot ".env.timeweb.local"))

$sshHost = $config["TIMEWEB_SSH_HOST"]
if ([string]::IsNullOrWhiteSpace($sshHost)) {
  $sshHost = $config["TIMEWEB_PUBLIC_IP"]
}

$sshUser = $config["TIMEWEB_SSH_USER"]
if ([string]::IsNullOrWhiteSpace($sshUser)) {
  $sshUser = "root"
}

$sshPort = $config["TIMEWEB_SSH_PORT"]
if ([string]::IsNullOrWhiteSpace($sshPort)) {
  $sshPort = "22"
}

$remoteDir = $config["TIMEWEB_REMOTE_DIR"]
if ([string]::IsNullOrWhiteSpace($remoteDir)) {
  $remoteDir = "/opt/TverActive"
}

if ([string]::IsNullOrWhiteSpace($sshHost)) {
  throw "TIMEWEB_SSH_HOST or TIMEWEB_PUBLIC_IP is required in $EnvFile"
}

$target = "${sshUser}@${sshHost}"
$archiveName = "timeweb-source.tar.gz"
$deployDir = Join-Path $repoRoot ".deploy"
$archivePath = Join-Path $deployDir $archiveName

New-Item -ItemType Directory -Force -Path $deployDir | Out-Null
if (Test-Path -LiteralPath $archivePath) {
  Remove-Item -LiteralPath $archivePath -Force
}

Write-Host "Packing project for $target..."
$tarArgs = @(
  "-czf", $archivePath,
  "--exclude=.deploy",
  "--exclude=.runtime-logs",
  "--exclude=.env.timeweb.local",
  "--exclude=**/.git",
  "--exclude=**/node_modules",
  "--exclude=**/.next",
  "--exclude=**/dist",
  "--exclude=at-admin-master",
  "--exclude=at-backend-main",
  "-C", $repoRoot,
  "."
)
& tar @tarArgs

if ($PackageOnly) {
  Write-Host "Package created at $archivePath"
  return
}

$quotedRemoteDir = Remote-Quote $remoteDir
$remoteDeployCommand = "mkdir -p $quotedRemoteDir && tar -xzf /tmp/$archiveName -C $quotedRemoteDir && cd $quotedRemoteDir && (chmod +x infra/*.sh max-bot-service/scripts/*.sh 2>/dev/null || true) && ./infra/deploy-timeweb.sh"
$password = $config["TIMEWEB_ROOT_PASSWORD"]
$docker = Get-Command docker -ErrorAction SilentlyContinue

if (-not [string]::IsNullOrWhiteSpace($password) -and $docker -and -not $NoDockerSshpass) {
  Write-Host "Uploading and deploying through Docker + sshpass..."
  & docker run --rm `
    -e "SSHPASS=$password" `
    -e "SSH_TARGET=$target" `
    -e "SSH_PORT=$sshPort" `
    -e "REMOTE_DEPLOY_COMMAND=$remoteDeployCommand" `
    -v "${repoRoot}:/work" `
    alpine:3.20 `
    sh -lc "apk add --no-cache openssh-client sshpass >/dev/null && sshpass -e ssh -o StrictHostKeyChecking=accept-new -p `"`$SSH_PORT`" `"`$SSH_TARGET`" 'mkdir -p /tmp' && sshpass -e scp -o StrictHostKeyChecking=accept-new -P `"`$SSH_PORT`" /work/.deploy/$archiveName `"`$SSH_TARGET`":/tmp/$archiveName && sshpass -e ssh -o StrictHostKeyChecking=accept-new -p `"`$SSH_PORT`" `"`$SSH_TARGET`" `"`$REMOTE_DEPLOY_COMMAND`""
} elseif (-not [string]::IsNullOrWhiteSpace($password) -and (Get-Module -ListAvailable Posh-SSH)) {
  Write-Host "Uploading and deploying through Posh-SSH..."
  Import-Module Posh-SSH

  $securePassword = ConvertTo-SecureString $password -AsPlainText -Force
  $credential = [System.Management.Automation.PSCredential]::new($sshUser, $securePassword)
  $sshSession = $null
  $sftpSession = $null
  $connectTimeout = 90
  $connectAttempts = 3

  try {
    for ($attempt = 1; $attempt -le $connectAttempts; $attempt++) {
      try {
        $sshSession = New-SSHSession -ComputerName $sshHost -Credential $credential -Port ([int]$sshPort) -AcceptKey -Force -ConnectionTimeout $connectTimeout
        break
      } catch {
        if ($attempt -eq $connectAttempts) {
          throw
        }
        Write-Warning "SSH connect attempt $attempt/$connectAttempts failed: $($_.Exception.Message). Retrying..."
        Start-Sleep -Seconds (3 * $attempt)
      }
    }

    for ($attempt = 1; $attempt -le $connectAttempts; $attempt++) {
      try {
        $sftpSession = New-SFTPSession -ComputerName $sshHost -Credential $credential -Port ([int]$sshPort) -AcceptKey -Force -ConnectionTimeout $connectTimeout -OperationTimeout 240
        break
      } catch {
        if ($attempt -eq $connectAttempts) {
          throw
        }
        Write-Warning "SFTP connect attempt $attempt/$connectAttempts failed: $($_.Exception.Message). Retrying..."
        Start-Sleep -Seconds (3 * $attempt)
      }
    }

    Set-SFTPItem -SessionId $sftpSession.SessionId -Path $archivePath -Destination "/tmp" -Force | Out-Null

    $result = Invoke-SSHCommand -SessionId $sshSession.SessionId -Command $remoteDeployCommand -TimeOut 3600
    if ($result.Output) {
      $result.Output | ForEach-Object { Write-Host $_ }
    }
    if ($result.Error) {
      $result.Error | ForEach-Object { Write-Error $_ }
    }
    if ($result.ExitStatus -ne 0) {
      throw "Remote deploy failed with exit status $($result.ExitStatus)"
    }
  } finally {
    if ($sftpSession) {
      Remove-SFTPSession -SessionId $sftpSession.SessionId | Out-Null
    }
    if ($sshSession) {
      Remove-SSHSession -SessionId $sshSession.SessionId | Out-Null
    }
  }
} else {
  if (-not [string]::IsNullOrWhiteSpace($password)) {
    Write-Warning "TIMEWEB_ROOT_PASSWORD is set, but Docker sshpass mode and Posh-SSH are unavailable or disabled. Native OpenSSH will ask for the password interactively."
  }

  Write-Host "Uploading and deploying through native OpenSSH..."
  & ssh -o StrictHostKeyChecking=accept-new -p $sshPort $target "mkdir -p /tmp"
  & scp -o StrictHostKeyChecking=accept-new -P $sshPort $archivePath "${target}:/tmp/$archiveName"
  & ssh -o StrictHostKeyChecking=accept-new -p $sshPort $target $remoteDeployCommand
}

Write-Host "Done."
