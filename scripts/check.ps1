$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent $PSScriptRoot

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

    Invoke-ExternalCheck -Label "Running Go tests" -Command "go" -Arguments @("test", "./...")
    Invoke-ExternalCheck -Label "Running Go vet" -Command "go" -Arguments @("vet", "./...")

    Push-Location "apps/miniapp"
    try {
        if (-not (Test-Path -LiteralPath "node_modules")) {
            Invoke-ExternalCheck -Label "Installing miniapp dependencies" -Command "npm" -Arguments @("ci")
        }
        Invoke-ExternalCheck -Label "Running miniapp tests" -Command "npm" -Arguments @("test", "--", "--run")
        Invoke-ExternalCheck -Label "Checking miniapp types" -Command "npm" -Arguments @("run", "type-check")
        Invoke-ExternalCheck -Label "Building WeChat miniapp" -Command "npm" -Arguments @("run", "build:mp-weixin")
    }
    finally {
        Pop-Location
    }

    Write-Host "All project checks passed."
}
finally {
    Pop-Location
}
