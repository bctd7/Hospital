param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("identity", "appointment", "guidance")]
    [string]$Service,

    [Parameter(Mandatory = $true)]
    [ValidateSet("up", "down", "version")]
    [string]$Direction,

    [ValidateRange(1, [int]::MaxValue)]
    [int]$Steps = 1
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
. (Join-Path (Split-Path -Parent $PSScriptRoot) "lib/environment.ps1")
Import-ProjectEnvironment -RepositoryRoot $repositoryRoot

$serviceConfiguration = switch ($Service) {
    "identity" {
        @{
            Migrations = Join-Path $repositoryRoot "migrations/identity"
            DsnEnvironment = "IDENTITY_MYSQL_DSN"
        }
    }
    "appointment" {
        @{
            Migrations = Join-Path $repositoryRoot "migrations/appointment"
            DsnEnvironment = "APPOINTMENT_MYSQL_DSN"
        }
    }
    "guidance" {
        @{
            Migrations = Join-Path $repositoryRoot "migrations/guidance"
            DsnEnvironment = "GUIDANCE_MYSQL_DSN"
        }
    }
}

$applicationEnvironment = [Environment]::GetEnvironmentVariable("APP_ENV", "Process")
if ($Direction -eq "down" -and $applicationEnvironment -notin @("local", "test", "ci")) {
    throw "Down migrations are restricted to APP_ENV=local, test, or ci. Production rollback requires a reviewed runbook."
}

Get-RequiredEnvironmentValue -Name $serviceConfiguration.DsnEnvironment | Out-Null

$arguments = @(
    "run",
    "./tools/database/migrate",
    "-migrations", $serviceConfiguration.Migrations,
    "-dsn-env", $serviceConfiguration.DsnEnvironment,
    "-direction", $Direction
)

if ($Direction -eq "down") {
    $arguments += @("-steps", $Steps)
}
Push-Location $repositoryRoot
try {
    & go @arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Database migration failed with exit code $LASTEXITCODE."
    }
}
finally {
    Pop-Location
}
