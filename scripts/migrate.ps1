param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("identity", "appointment", "guidance")]
    [string]$Service,

    [Parameter(Mandatory = $true)]
    [ValidateSet("up", "down", "version", "baseline")]
    [string]$Direction,

    [ValidateRange(1, [int]::MaxValue)]
    [int]$Steps = 1,

    [ValidateRange(1, [int]::MaxValue)]
    [int]$BaselineVersion = 1,

    [switch]$AcknowledgeExistingSchema
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent $PSScriptRoot
. (Join-Path $PSScriptRoot "lib/environment.ps1")
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

if ($Direction -eq "baseline" -and -not $AcknowledgeExistingSchema) {
    throw "Baseline records an existing schema without running SQL. Re-run with -AcknowledgeExistingSchema only after verifying the database matches the migration files."
}
if ($Direction -eq "baseline") {
    $baselinePattern = "{0:D6}_*.up.sql" -f $BaselineVersion
    $baselineFiles = @(Get-ChildItem -LiteralPath $serviceConfiguration.Migrations -Filter $baselinePattern)
    if ($baselineFiles.Count -ne 1) {
        throw "Baseline version $BaselineVersion does not identify exactly one migration in $($serviceConfiguration.Migrations)."
    }
}

$applicationEnvironment = [Environment]::GetEnvironmentVariable("APP_ENV", "Process")
if ($Direction -eq "down" -and $applicationEnvironment -notin @("local", "test", "ci")) {
    throw "Down migrations are restricted to APP_ENV=local, test, or ci. Production rollback requires a reviewed runbook."
}

Get-RequiredEnvironmentValue -Name $serviceConfiguration.DsnEnvironment | Out-Null

$arguments = @(
    "run",
    "./tools/db-migrate",
    "-migrations", $serviceConfiguration.Migrations,
    "-dsn-env", $serviceConfiguration.DsnEnvironment,
    "-direction", $Direction
)

if ($Direction -eq "down") {
    $arguments += @("-steps", $Steps)
}
if ($Direction -eq "baseline") {
    $arguments += @("-baseline-version", $BaselineVersion)
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
