$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$composeFile = Join-Path $repositoryRoot "deploy/compose/docker-compose.yml"
$environmentFile = Join-Path $repositoryRoot ".env"

. (Join-Path (Split-Path -Parent $PSScriptRoot) "lib/environment.ps1")
Import-ProjectEnvironment -RepositoryRoot $repositoryRoot

$applicationEnvironment = Get-RequiredEnvironmentValue -Name "APP_ENV"
if ($applicationEnvironment -ne "local") {
    throw "bootstrap-local.ps1 is restricted to APP_ENV=local. Current value: $applicationEnvironment"
}
if (-not (Test-Path -LiteralPath $environmentFile)) {
    throw "Local bootstrap requires $environmentFile. Copy .env.example to .env and review its values first."
}

$identityUser = Get-RequiredEnvironmentValue -Name "IDENTITY_MYSQL_USER"
$identityPassword = Get-RequiredEnvironmentValue -Name "IDENTITY_MYSQL_PASSWORD"
$appointmentUser = Get-RequiredEnvironmentValue -Name "APPOINTMENT_MYSQL_USER"
$appointmentPassword = Get-RequiredEnvironmentValue -Name "APPOINTMENT_MYSQL_PASSWORD"
$guidanceUser = Get-RequiredEnvironmentValue -Name "GUIDANCE_MYSQL_USER"
$guidancePassword = Get-RequiredEnvironmentValue -Name "GUIDANCE_MYSQL_PASSWORD"
if ($identityUser -notmatch "^[A-Za-z0-9_]+$") {
    throw "IDENTITY_MYSQL_USER contains unsupported characters."
}
if ($identityPassword -notmatch "^[A-Za-z0-9_.@%+=:-]+$") {
    throw "For local bootstrap, IDENTITY_MYSQL_PASSWORD may only contain letters, numbers, and ._@%+=:- characters."
}
if ($appointmentUser -notmatch "^[A-Za-z0-9_]+$") {
    throw "APPOINTMENT_MYSQL_USER contains unsupported characters."
}
if ($appointmentPassword -notmatch "^[A-Za-z0-9_.@%+=:-]+$") {
    throw "For local bootstrap, APPOINTMENT_MYSQL_PASSWORD may only contain letters, numbers, and ._@%+=:- characters."
}
if ($guidanceUser -notmatch "^[A-Za-z0-9_]+$") {
    throw "GUIDANCE_MYSQL_USER contains unsupported characters."
}
if ($guidancePassword -notmatch "^[A-Za-z0-9_.@%+=:-]+$") {
    throw "For local bootstrap, GUIDANCE_MYSQL_PASSWORD may only contain letters, numbers, and ._@%+=:- characters."
}

$composeArguments = @(
    "compose",
    "--env-file", $environmentFile,
    "-f", $composeFile
)

Push-Location $repositoryRoot
try {
    & docker @composeArguments up -d mysql
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to start the local MySQL container."
    }

    $ready = $false
    for ($attempt = 1; $attempt -le 30; $attempt++) {
        & docker @composeArguments exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqladmin ping --protocol=socket -uroot --silent' *> $null
        if ($LASTEXITCODE -eq 0) {
            $ready = $true
            break
        }
        Start-Sleep -Seconds 2
    }
    if (-not $ready) {
        throw "MySQL did not become ready within 60 seconds."
    }

    $bootstrapSQL = @"
CREATE DATABASE IF NOT EXISTS hospital_identity
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_0900_ai_ci;
CREATE USER IF NOT EXISTS '$identityUser'@'%'
    IDENTIFIED BY '$identityPassword';
GRANT ALL PRIVILEGES ON hospital_identity.* TO '$identityUser'@'%';
CREATE DATABASE IF NOT EXISTS hospital_appointment
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_0900_ai_ci;
CREATE USER IF NOT EXISTS '$appointmentUser'@'%'
    IDENTIFIED BY '$appointmentPassword';
GRANT ALL PRIVILEGES ON hospital_appointment.* TO '$appointmentUser'@'%';
CREATE DATABASE IF NOT EXISTS hospital_guidance
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_0900_ai_ci;
CREATE USER IF NOT EXISTS '$guidanceUser'@'%'
    IDENTIFIED BY '$guidancePassword';
GRANT ALL PRIVILEGES ON hospital_guidance.* TO '$guidanceUser'@'%';
FLUSH PRIVILEGES;
"@

    $bootstrapSQL | & docker @composeArguments exec -T mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot'
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to create local service databases and users."
    }

    & (Join-Path $PSScriptRoot "migrate.ps1") -Service identity -Direction up
    if ($LASTEXITCODE -ne 0) {
        throw "Identity migration failed after local database bootstrap."
    }

    & (Join-Path $PSScriptRoot "migrate.ps1") -Service appointment -Direction up
    if ($LASTEXITCODE -ne 0) {
        throw "Appointment migration failed after local database bootstrap."
    }

    & (Join-Path $PSScriptRoot "migrate.ps1") -Service guidance -Direction up
    if ($LASTEXITCODE -ne 0) {
        throw "Guidance migration failed after local database bootstrap."
    }

    Write-Host "Local Identity, Appointment, and Guidance databases are ready."
}
finally {
    Pop-Location
}
