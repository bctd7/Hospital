$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

function Invoke-ExternalCheck {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Label,

        [Parameter(Mandatory = $true)]
        [string]$Command,

        [string[]]$Arguments = @()
    )

    Write-Host "==> $Label"
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Label failed with exit code $LASTEXITCODE."
    }
}

Push-Location $repositoryRoot
try {
    Write-Host "==> Validating Protobuf contracts"
    & (Join-Path $repositoryRoot "scripts/contracts/generate-protobuf.ps1") -Check

    Write-Host "==> Validating flattened database migrations"
    & (Join-Path $repositoryRoot "scripts/database/validate-flat-migrations.ps1")

    Invoke-ExternalCheck -Label "Validating go-zero API contract" -Command "goctl" -Arguments @(
        "api", "validate", "-api", "contracts/api/app.api"
    )

    Write-Host "==> Validating event envelope JSON"
    Get-Content -Raw -Encoding UTF8 "contracts/events/event-envelope.schema.json" |
        ConvertFrom-Json |
        Out-Null

    Invoke-ExternalCheck -Label "Validating Docker Compose configuration" -Command "docker" -Arguments @(
        "compose",
        "--env-file", ".env.example",
        "-f", "deploy/compose/docker-compose.yml",
        "--profile", "messaging",
        "config", "--quiet"
    )

    $previousProductionEnvFile = $env:HOSPITAL_PRODUCTION_ENV_FILE
    try {
        $env:HOSPITAL_PRODUCTION_ENV_FILE = "env.example"
        Invoke-ExternalCheck -Label "Validating production Docker Compose configuration" -Command "docker" -Arguments @(
            "compose",
            "--env-file", "deploy/production/env.example",
            "-f", "deploy/production/docker-compose.yml",
            "config", "--quiet"
        )
    }
    finally {
        if ($null -eq $previousProductionEnvFile) {
            Remove-Item Env:HOSPITAL_PRODUCTION_ENV_FILE -ErrorAction SilentlyContinue
        }
        else {
            $env:HOSPITAL_PRODUCTION_ENV_FILE = $previousProductionEnvFile
        }
    }

    Invoke-ExternalCheck -Label "Running Go tests" -Command "go" -Arguments @("test", "./...")
    Invoke-ExternalCheck -Label "Running Go vet" -Command "go" -Arguments @("vet", "./...")

    Push-Location "apps/miniapp"
    try {
        if (-not (Test-Path -LiteralPath "node_modules")) {
            Invoke-ExternalCheck -Label "Installing miniapp dependencies" -Command "npm" -Arguments @("ci")
        }
        Invoke-ExternalCheck -Label "Running miniapp tests" -Command "npm" -Arguments @("test", "--", "--run")
        Invoke-ExternalCheck -Label "Checking miniapp types" -Command "npm" -Arguments @("run", "type-check")
        Invoke-ExternalCheck -Label "Building development WeChat miniapp" -Command "npm" -Arguments @("run", "build:dev:mp-weixin")
        Invoke-ExternalCheck -Label "Building release WeChat miniapp" -Command "npm" -Arguments @("run", "build:mp-weixin")
    }
    finally {
        Pop-Location
    }

    Write-Host "All project checks passed."
}
finally {
    Pop-Location
}
